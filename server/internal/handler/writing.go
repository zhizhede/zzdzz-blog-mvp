package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

// WritingHandler AI 写作工作流: compose 单步接口 + 风格卡 + 版本快照.
// 与 AIHandler 有意分开: 写作流不是聊天, 无会话持久化; 版本接口挂在
// /articles/:id/versions 下但复用本 handler, 因为快照的生命周期属于写作流.
type WritingHandler struct {
	cfg *config.AIConfig
	svc *service.WritingService
}

func NewWritingHandler(cfg *config.AIConfig, svc *service.WritingService) *WritingHandler {
	return &WritingHandler{cfg: cfg, svc: svc}
}

func (h *WritingHandler) aiConfigured() bool {
	return h.cfg.Enabled && h.cfg.APIKey != "" && h.cfg.BaseURL != "" && h.cfg.Model != ""
}

func (h *WritingHandler) openaiClient() *openai.Client {
	cfg := openai.DefaultConfig(h.cfg.APIKey)
	cfg.BaseURL = h.cfg.BaseURL
	return openai.NewClientWithConfig(cfg)
}

type composeReq struct {
	Action      string `json:"action" binding:"required,oneof=outline draft refine"`
	Material    string `json:"material"`
	Outline     string `json:"outline"`
	Draft       string `json:"draft"`
	Instruction string `json:"instruction"`
	// UseOwnStyle 请求注入风格卡; 查不到时后端降级为通用文风并推 meta 事件(设计 §4.3/§5.3)
	UseOwnStyle bool `json:"use_own_style"`
}

// Compose POST /api/v1/ai/compose
// 单步生成: 前端把当前大纲/稿件随请求回传, 后端调一次 LLM 并以 SSE 流式返回.
// 事件序列: [sources] [meta] delta... [DONE], 与 AI 会话同一套格式.
func (h *WritingHandler) Compose(c *gin.Context) {
	if !h.aiConfigured() {
		response.ServerError(c, "AI 服务未配置(需在 config.yaml 设置 ai.api_key / ai.base_url / ai.model)")
		return
	}
	var req composeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// 按动作校验必填段(设计 §4.4); 长度按 rune 计, 与文档「字」口径一致
	switch action := service.ComposeAction(req.Action); action {
	case service.ComposeOutline, service.ComposeDraft:
		if utf8.RuneCountInString(req.Material) == 0 {
			response.BadRequest(c, "请提供素材")
			return
		}
		if utf8.RuneCountInString(req.Material) > service.MaxMaterialLen {
			response.BadRequest(c, fmt.Sprintf("material too long (max %d 字)", service.MaxMaterialLen))
			return
		}
	case service.ComposeRefine:
		if utf8.RuneCountInString(req.Draft) == 0 {
			response.BadRequest(c, "请提供草稿")
			return
		}
		if utf8.RuneCountInString(req.Draft) > service.MaxDraftLen {
			response.BadRequest(c, fmt.Sprintf("draft too long (max %d 字)", service.MaxDraftLen))
			return
		}
		if strings.TrimSpace(req.Instruction) == "" {
			response.BadRequest(c, "润色需要提供指令")
			return
		}
	}
	if n := utf8.RuneCountInString(req.Outline); n > service.MaxDraftLen {
		response.BadRequest(c, "提纲过长")
		return
	}
	if n := utf8.RuneCountInString(req.Instruction); n > 2000 {
		response.BadRequest(c, "指令过长(最多 2000 字)")
		return
	}

	// 风格卡注入(§5.3): profile 存在 → 拼 system + 推 sources(参考了哪几篇);
	// 请求要但查不到 → 推 meta 提示前端降级, 行为退化为通用文风.
	var styleCard string
	var samples []model.Article
	uid := userIDOf(c)
	if req.UseOwnStyle && h.svc != nil {
		if p, err := h.svc.GetStyleProfile(uid); err == nil && p != nil && strings.TrimSpace(p.Profile) != "" {
			styleCard = p.Profile
			samples, _ = h.svc.StyleSampleArticles(uid)
		}
	}

	msgs := service.BuildComposeMessages(service.ComposeInput{
		Action:      service.ComposeAction(req.Action),
		Material:    req.Material,
		Outline:     req.Outline,
		Draft:       req.Draft,
		Instruction: req.Instruction,
		StyleCard:   styleCard,
	})

	// SSE 头与事件格式与 ai.go 一致: {"delta"} / {"sources"} / {"meta"} / {"error"} / [DONE]
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeSSEError(c.Writer, errors.New("streaming unsupported"))
		return
	}
	writeSSEPayload := func(v gin.H) {
		payload, _ := json.Marshal(v)
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
	}

	if req.UseOwnStyle {
		if styleCard != "" && len(samples) > 0 {
			srcs := make([]gin.H, 0, len(samples))
			for _, a := range samples {
				srcs = append(srcs, gin.H{"id": a.ID, "title": a.Title})
			}
			writeSSEPayload(gin.H{"sources": srcs})
		} else {
			writeSSEPayload(gin.H{"meta": gin.H{"style_missing": true}})
		}
	}

	stream, err := h.openaiClient().CreateChatCompletionStream(c.Request.Context(), openai.ChatCompletionRequest{
		Model:       h.cfg.Model,
		Messages:    msgs,
		Stream:      true,
		Temperature: service.Temperature(service.ComposeAction(req.Action)),
	})
	if err != nil {
		writeSSEError(c.Writer, err)
		flusher.Flush()
		return
	}
	defer stream.Close()

	stripper := service.NewFenceStripper()
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 收尾: 挂起的尾部围栏此时才能判定, 若有正文补发一段
			if tail := stripper.Finish(); tail != "" {
				writeSSEPayload(gin.H{"delta": tail})
			}
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
		if err != nil {
			writeSSEError(c.Writer, err)
			flusher.Flush()
			return
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if out := stripper.Write(delta); out != "" {
			writeSSEPayload(gin.H{"delta": out})
		}
	}
}

// -------------------- 风格卡(设计 §5) --------------------

// GetStyleProfile GET /api/v1/writing/style-profile; 无风格卡时 data 为 null
func (h *WritingHandler) GetStyleProfile(c *gin.Context) {
	p, err := h.svc.GetStyleProfile(userIDOf(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, p)
}

type saveProfileReq struct {
	Profile string `json:"profile" binding:"required,max=2000"`
}

// PutStyleProfile PUT /api/v1/writing/style-profile; 手改, source 置 manual
func (h *WritingHandler) PutStyleProfile(c *gin.Context) {
	var req saveProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.SaveManualStyleProfile(userIDOf(c), req.Profile)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, p)
}

// DeriveStyleProfile POST /api/v1/writing/style-profile/derive; 同步提炼(低频操作)
func (h *WritingHandler) DeriveStyleProfile(c *gin.Context) {
	p, samples, err := h.svc.DeriveStyleProfile(c.Request.Context(), userIDOf(c))
	if err != nil {
		if errors.Is(err, service.ErrNoStyleSamples) || errors.Is(err, service.ErrDeriveTooFrequent) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	type sampleItem struct {
		ID    uint64 `json:"id"`
		Title string `json:"title"`
	}
	items := make([]sampleItem, 0, len(samples))
	for _, a := range samples {
		items = append(items, sampleItem{ID: a.ID, Title: a.Title})
	}
	response.OK(c, gin.H{"profile": p, "samples": items})
}

// -------------------- 版本快照(设计 §6) --------------------

// CreateVersion POST /api/v1/articles/:id/versions {origin, note}
// 存文章当前内容为快照; 采用 AI 结果前调用, 失败由前端阻断采用.
func (h *WritingHandler) CreateVersion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req struct {
		Origin string `json:"origin"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	v, err := h.svc.CreateSnapshot(id, req.Origin, req.Note, actorOf(c))
	if err != nil {
		writeWritingErr(c, err)
		return
	}
	response.OK(c, v)
}

// ListVersions GET /api/v1/articles/:id/versions; 列表不含 content
func (h *WritingHandler) ListVersions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的 ID")
		return
	}
	vs, err := h.svc.ListVersions(id, actorOf(c))
	if err != nil {
		writeWritingErr(c, err)
		return
	}
	response.OK(c, vs)
}

// GetVersion GET /api/v1/articles/:id/versions/:vid; 含 content, 供回填预览
func (h *WritingHandler) GetVersion(c *gin.Context) {
	id, vid, ok := versionParams(c)
	if !ok {
		return
	}
	v, err := h.svc.GetVersion(id, vid, actorOf(c))
	if err != nil {
		writeWritingErr(c, err)
		return
	}
	response.OK(c, v)
}

// RestoreVersion POST /api/v1/articles/:id/versions/:vid/restore
// 先把当前内容存 pre_restore 快照, 再把目标版本写回文章(§6.3)
func (h *WritingHandler) RestoreVersion(c *gin.Context) {
	id, vid, ok := versionParams(c)
	if !ok {
		return
	}
	a, err := h.svc.RestoreVersion(id, vid, actorOf(c))
	if err != nil {
		writeWritingErr(c, err)
		return
	}
	response.OK(c, a)
}

func versionParams(c *gin.Context) (id, vid uint64, ok bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的 ID")
		return 0, 0, false
	}
	vid, err = strconv.ParseUint(c.Param("vid"), 10, 64)
	if err != nil || vid == 0 {
		response.BadRequest(c, "无效的版本 ID")
		return 0, 0, false
	}
	return id, vid, true
}

func writeWritingErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrArticleNotFound), errors.Is(err, service.ErrVersionNotFound):
		response.Fail(c, 404, 4004, err.Error())
	case errors.Is(err, service.ErrArticleNotOwned):
		response.Fail(c, 403, 4003, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}
