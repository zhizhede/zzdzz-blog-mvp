package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

type AIHandler struct {
	cfg *config.AIConfig
	svc *service.AIService
	// recall AI 回顾服务, 可为 nil(未启用); Enabled() 内部判空
	recall *service.RecallService
	// visits 访问日志服务(0016), 为 admin 对话注入实时访问摘要; 可为 nil
	visits *service.VisitLogService
}

func NewAIHandler(cfg *config.AIConfig, svc *service.AIService, recall *service.RecallService, visits *service.VisitLogService) *AIHandler {
	return &AIHandler{cfg: cfg, svc: svc, recall: recall, visits: visits}
}

// -------------------- 会话管理 --------------------

func (h *AIHandler) ListConversations(c *gin.Context) {
	uid := userIDOf(c)
	items, err := h.svc.ListConversations(uid)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (h *AIHandler) CreateConversation(c *gin.Context) {
	uid := userIDOf(c)
	conv, err := h.svc.CreateConversation(uid)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, conv)
}

type renameConvReq struct {
	Title string `json:"title" binding:"required,min=1,max=100"`
}

func (h *AIHandler) RenameConversation(c *gin.Context) {
	uid := userIDOf(c)
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req renameConvReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.RenameConversation(uid, convID, req.Title); err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.Fail(c, 404, 4004, "会话不存在")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AIHandler) DeleteConversation(c *gin.Context) {
	uid := userIDOf(c)
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	if err := h.svc.DeleteConversation(uid, convID); err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.Fail(c, 404, 4004, "会话不存在")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AIHandler) ListMessages(c *gin.Context) {
	uid := userIDOf(c)
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	msgs, err := h.svc.GetRecentMessages(uid, convID, limit)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.Fail(c, 404, 4004, "会话不存在")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, msgs)
}

// -------------------- 发送消息（流式 + 持久化）--------------------

type sendMessageReq struct {
	Content string `json:"content" binding:"required,min=1"`
}

// SendMessage 是持久化版的核心入口:
//  1. 校验会话属于当前用户
//  2. 把 user 消息入库
//  3. 从 DB 取最近 20 条历史构造上下文
//  4. 流式调 LLM,边收 chunk 边 UPDATE 数据库
//  5. 流式结束:Finalize(标题/updated_at)
func (h *AIHandler) SendMessage(c *gin.Context) {
	if !h.aiConfigured() {
		response.ServerError(c, "AI 服务未配置(需在 config.yaml 设置 ai.api_key / ai.base_url / ai.model)")
		return
	}
	uid := userIDOf(c)
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req sendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入内容")
		return
	}
	if _, err := h.svc.GetConversation(uid, convID); err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			response.Fail(c, 404, 4004, "会话不存在")
			return
		}
		response.ServerError(c, err.Error())
		return
	}

	// 1. user 消息入库
	if _, err := h.svc.AppendUserMessage(convID, req.Content); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	// 2. 加载最近 20 条作为上下文
	histMsgs, err := h.svc.GetRecentMessages(uid, convID, 20)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	llmMsgs := make([]openai.ChatCompletionMessage, 0, len(histMsgs))
	for _, m := range histMsgs {
		llmMsgs = append(llmMsgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// 2.5 AI 回顾: 按可见性检索用户写过的内容, 无感注入上下文.
	// 任何失败都静默降级为普通对话, 不影响主流程(设计见 doc/v0.3-tech-design.md §6).
	var recallSources []service.RecallSource
	if h.recall.Enabled() {
		adminFlag := false
		if v, ok := c.Get("is_admin"); ok {
			adminFlag, _ = v.(bool)
		}
		prompt, sources, err := h.recall.RecallForQuery(
			c.Request.Context(),
			service.AccessQuery{UserID: uid, IsAdmin: adminFlag},
			req.Content,
		)
		if err != nil {
			log.Printf("[recall] search failed, degrade to plain chat: %v", err)
		} else if len(sources) > 0 {
			llmMsgs = append([]openai.ChatCompletionMessage{
				{Role: "system", Content: prompt},
			}, llmMsgs...)
			recallSources = sources
		}
	}

	// 2.6 访问统计注入(0016): 仅 admin 对话注入实时访问摘要, AI 可回答
	// "今天多少人访问/最近谁来了"类问题; 普通用户不注入(访问数据含 IP, 不外泄).
	// 查询失败静默降级为普通对话, 不影响主流程.
	if v, ok := c.Get("is_admin"); ok {
		if adminFlag, _ := v.(bool); adminFlag && h.visits != nil {
			if summary, err := h.visits.StatsForAI(); err == nil && summary != "" {
				llmMsgs = append([]openai.ChatCompletionMessage{
					{Role: "system", Content: summary},
				}, llmMsgs...)
			} else if err != nil {
				log.Printf("[visit] stats summary for ai failed: %v", err)
			}
		}
	}

	// 3. assistant 占位行
	assistantID, err := h.svc.CreateAssistantPlaceholder(convID)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	// 4. SSE 流式
	h.streamChatWithPersist(c, uid, convID, assistantID, llmMsgs, recallSources)
}

// -------------------- 旧接口（保留兼容）--------------------

type chatMessage struct {
	Role    string `json:"role" binding:"required,oneof=system user assistant"`
	Content string `json:"content" binding:"required"`
}

type chatReq struct {
	Messages []chatMessage `json:"messages" binding:"required,min=1,dive"`
	Stream   bool          `json:"stream"`
}

// Chat 是无状态版,继续保留供前端尚未升级时使用。
func (h *AIHandler) Chat(c *gin.Context) {
	if !h.aiConfigured() {
		response.ServerError(c, "AI 服务未配置(需在 config.yaml 设置 ai.api_key / ai.base_url / ai.model)")
		return
	}

	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "messages 格式应为 [{role, content}]")
		return
	}

	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	client := h.openaiClient()

	if req.Stream {
		h.streamChat(c, client, msgs)
		return
	}

	resp, err := client.CreateChatCompletion(c.Request.Context(), openai.ChatCompletionRequest{
		Model:    h.cfg.Model,
		Messages: msgs,
	})
	if err != nil {
		response.ServerError(c, fmt.Sprintf("upstream error: %v", err))
		return
	}
	if len(resp.Choices) == 0 {
		response.ServerError(c, "AI 服务未返回内容")
		return
	}
	response.OK(c, gin.H{
		"message": resp.Choices[0].Message.Content,
		"usage":   resp.Usage,
		"model":   resp.Model,
	})
}

// -------------------- 内部工具 --------------------

func (h *AIHandler) aiConfigured() bool {
	return h.cfg.Enabled && h.cfg.APIKey != "" && h.cfg.BaseURL != "" && h.cfg.Model != ""
}

func (h *AIHandler) openaiClient() *openai.Client {
	cfg := openai.DefaultConfig(h.cfg.APIKey)
	cfg.BaseURL = h.cfg.BaseURL
	return openai.NewClientWithConfig(cfg)
}

// streamChat 旧版无状态 SSE。
func (h *AIHandler) streamChat(c *gin.Context, client *openai.Client, msgs []openai.ChatCompletionMessage) {
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

	stream, err := client.CreateChatCompletionStream(c.Request.Context(), openai.ChatCompletionRequest{
		Model:    h.cfg.Model,
		Messages: msgs,
		Stream:   true,
	})
	if err != nil {
		writeSSEError(c.Writer, err)
		flusher.Flush()
		return
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			writeSSEDone(c.Writer)
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
		payload, _ := json.Marshal(gin.H{"delta": delta})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
	}
}

// streamChatWithPersist 流式版,每个 chunk 同时:
//   - 推 SSE 给前端
//   - UPDATE 数据库(content || delta)
func (h *AIHandler) streamChatWithPersist(c *gin.Context, userID, convID, assistantID uint64, msgs []openai.ChatCompletionMessage, recallSources []service.RecallSource) {
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

	// 引用来源作为首个 SSE 事件推给前端, 早于所有 delta.
	// 前端按 JSON 字段分支解析, 旧前端不认识 sources 字段会自然忽略, 向后兼容.
	if len(recallSources) > 0 {
		payload, _ := json.Marshal(gin.H{"sources": recallSources})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
	}

	client := h.openaiClient()
	stream, err := client.CreateChatCompletionStream(c.Request.Context(), openai.ChatCompletionRequest{
		Model:    h.cfg.Model,
		Messages: msgs,
		Stream:   true,
	})
	if err != nil {
		writeSSEError(c.Writer, err)
		flusher.Flush()
		return
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 流式结束:finalize
			if finErr := h.svc.FinalizeAssistant(userID, convID, assistantID); finErr != nil && !errors.Is(finErr, gorm.ErrRecordNotFound) {
				writeSSEError(c.Writer, finErr)
			} else {
				writeSSEDone(c.Writer)
			}
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
		// 先入库,再推送给前端;即使前端断了,数据库里也已收到
		if dbErr := h.svc.AppendDelta(assistantID, delta); dbErr != nil {
			writeSSEError(c.Writer, dbErr)
			flusher.Flush()
			return
		}
		payload, _ := json.Marshal(gin.H{"delta": delta})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
	}
}

// model.AIMessage 用于在 json 输出里跟前端类型对齐(若以后需要进一步过滤字段)。
var _ = model.AIMessage{}

func writeSSEError(w io.Writer, err error) {
	payload, _ := json.Marshal(gin.H{"error": err.Error()})
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func writeSSEDone(w io.Writer) {
	fmt.Fprintf(w, "data: [DONE]\n\n")
}