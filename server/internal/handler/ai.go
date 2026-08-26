package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/pkg/response"
)

type AIHandler struct {
	cfg *config.AIConfig
}

func NewAIHandler(cfg *config.AIConfig) *AIHandler {
	return &AIHandler{cfg: cfg}
}

type chatMessage struct {
	Role    string `json:"role" binding:"required,oneof=system user assistant"`
	Content string `json:"content" binding:"required"`
}

type chatReq struct {
	Messages []chatMessage `json:"messages" binding:"required,min=1,dive"`
	Stream   bool          `json:"stream"`
}

func (h *AIHandler) Chat(c *gin.Context) {
	if !h.cfg.Enabled || h.cfg.APIKey == "" || h.cfg.BaseURL == "" || h.cfg.Model == "" {
		response.ServerError(c, "AI not configured (set ai.api_key / ai.base_url / ai.model in config.yaml)")
		return
	}

	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "messages required: [{role, content}]")
		return
	}

	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	clientConfig := openai.DefaultConfig(h.cfg.APIKey)
	clientConfig.BaseURL = h.cfg.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

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
		response.ServerError(c, "no choices returned")
		return
	}
	response.OK(c, gin.H{
		"message": resp.Choices[0].Message.Content,
		"usage":   resp.Usage,
		"model":   resp.Model,
	})
}

func (h *AIHandler) streamChat(c *gin.Context, client *openai.Client, msgs []openai.ChatCompletionMessage) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		errors.New("streaming unsupported")
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

func writeSSEError(w io.Writer, err error) {
	payload, _ := json.Marshal(gin.H{"error": err.Error()})
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func writeSSEDone(w io.Writer) {
	fmt.Fprintf(w, "data: [DONE]\n\n")
}