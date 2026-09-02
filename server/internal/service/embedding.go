package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// EmbeddingClient 把文本批量转成向量. 这是供应商接缝: 未来换本地 ollama、
// 加缓存层, 只新增实现不改调用方, 见 doc/v0.3-tech-design.md §5.1.
//
// Embed 用于内容入库方向, EmbedQuery 用于检索查询方向:
// 非对称模型(如 MiniMax embo-01 的 db/query)两者参数不同, 对称模型实现成一样即可.
type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// OpenAIEmbedding 走 /embeddings 协议, base_url 来自 ai.embedding 配置,
// 可与 chat 用不同供应商. asymmetric=true 时(MiniMax embo-01)额外携带 type=db/query.
//
// style 两种取值:
//   - "openai"(默认): 请求 {"model","input"}, 响应 data[].embedding
//   - "minimax": MiniMax 原生格式, 请求 {"model","texts"(,type)}, 响应 vectors + base_resp.
//     实测 api.minimaxi.com/v1/embeddings 不认 OpenAI 的 input 字段, 必须适配.
//
// 不走 go-openai 的 CreateEmbeddings 是因为它的请求结构体塞不进 MiniMax 的
// texts/type 字段, 协议本身只是普通 JSON POST, 手写请求体反而通用.
type OpenAIEmbedding struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	dims       int
	asymmetric bool
	style      string
}

// NewOpenAIEmbedding 配置不齐返回错误(router 启动时 fail fast);
// dims 为 0 视作 1024(与迁移 0009 的 vector 列一致).
func NewOpenAIEmbedding(baseURL, apiKey, model string, dims int, asymmetric bool, style string) (*OpenAIEmbedding, error) {
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, fmt.Errorf("embedding not configured (need ai.embedding.base_url / api_key / model)")
	}
	if dims == 0 {
		dims = vectorDims
	}
	if dims != vectorDims {
		return nil, fmt.Errorf("ai.embedding.dims = %d != %d (migrations/0009 的 vector 列维度), 换维度需新迁移 + embed-backfill 全量重嵌", dims, vectorDims)
	}
	if style == "" {
		style = "openai"
	}
	if style != "openai" && style != "minimax" {
		return nil, fmt.Errorf("ai.embedding.style 只支持 openai / minimax, got %q", style)
	}
	return &OpenAIEmbedding{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		dims:       dims,
		asymmetric: asymmetric,
		style:      style,
	}, nil
}

// Embed 文档方向入库嵌入(索引文章/笔记时调用).
func (e *OpenAIEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embType := ""
	if e.asymmetric {
		embType = "db"
	}
	return e.embed(ctx, texts, embType)
}

// EmbedQuery 检索方向查询嵌入(用户消息召回时调用).
func (e *OpenAIEmbedding) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	embType := ""
	if e.asymmetric {
		embType = "query"
	}
	vecs, err := e.embed(ctx, []string{text}, embType)
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *OpenAIEmbedding) embed(ctx context.Context, texts []string, embType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	var body map[string]any
	if e.style == "minimax" {
		body = map[string]any{"model": e.model, "texts": texts}
		if embType != "" {
			body["type"] = embType
		}
	} else {
		body = map[string]any{"model": e.model, "input": texts}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding upstream: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding upstream http %d: %s", resp.StatusCode, truncateForLog(raw))
	}

	// 解析按 style 走; 数据为空时把原文带进错误, MiniMax 出错时常是 http 200 + base_resp 报错
	var vecs [][]float32
	if e.style == "minimax" {
		var parsed struct {
			Vectors  [][]float32 `json:"vectors"`
			BaseResp struct {
				StatusCode int    `json:"status_code"`
				StatusMsg  string `json:"status_msg"`
			} `json:"base_resp"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("embedding upstream bad json: %w: %s", err, truncateForLog(raw))
		}
		if parsed.BaseResp.StatusCode != 0 {
			return nil, fmt.Errorf("embedding upstream error %d: %s", parsed.BaseResp.StatusCode, parsed.BaseResp.StatusMsg)
		}
		vecs = parsed.Vectors
	} else {
		var parsed struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("embedding upstream bad json: %w: %s", err, truncateForLog(raw))
		}
		vecs = make([][]float32, len(parsed.Data))
		for i, d := range parsed.Data {
			vecs[i] = d.Embedding
		}
	}
	if len(vecs) != len(texts) {
		return nil, fmt.Errorf("embedding upstream returned %d vectors for %d inputs: %s",
			len(vecs), len(texts), truncateForLog(raw))
	}
	out := make([][]float32, len(texts))
	for i, v := range vecs {
		if len(v) != e.dims {
			return nil, fmt.Errorf("embedding dim mismatch: got %d, want %d (check ai.embedding.model)", len(v), e.dims)
		}
		out[i] = v
	}
	return out, nil
}

func truncateForLog(b []byte) string {
	s := string(b)
	if len(s) > 300 {
		s = s[:300] + "..."
	}
	return s
}

// vectorLiteral 把向量编码成 pgvector 字面量 '[1,2,3]'. 只由 float 拼接, 无注入面,
// 以参数传入 SQL 时再 ::vector 转型.
func vectorLiteral(vec []float32) string {
	var b strings.Builder
	b.Grow(len(vec) * 8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
