package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

// QuickPhraseHandler 快捷用语板(0019): 每用户的编辑器常用短语 CRUD.
// 路由全部挂 RequireAuth, 用户身份一律取自 token(userIDOf),
// 请求体不参与属主判定, 防止伪造 user_id 读写他人短语.
type QuickPhraseHandler struct {
	svc *service.QuickPhraseService
}

func NewQuickPhraseHandler(svc *service.QuickPhraseService) *QuickPhraseHandler {
	return &QuickPhraseHandler{svc: svc}
}

// List GET /api/v1/quick-phrases
func (h *QuickPhraseHandler) List(c *gin.Context) {
	ps, err := h.svc.List(userIDOf(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, ps)
}

type quickPhraseReq struct {
	// 内容校验(非空/长度)在 service 层做, 那里的报错文案对用户更友好
	Content string `json:"content"`
}

// Create POST /api/v1/quick-phrases
func (h *QuickPhraseHandler) Create(c *gin.Context) {
	var req quickPhraseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.Create(userIDOf(c), req.Content)
	if err != nil {
		writeQuickPhraseErr(c, err)
		return
	}
	response.OK(c, p)
}

// Update PUT /api/v1/quick-phrases/:id
func (h *QuickPhraseHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req quickPhraseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.Update(userIDOf(c), id, req.Content)
	if err != nil {
		writeQuickPhraseErr(c, err)
		return
	}
	response.OK(c, p)
}

// Delete DELETE /api/v1/quick-phrases/:id
func (h *QuickPhraseHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的 ID")
		return
	}
	if err := h.svc.Delete(userIDOf(c), id); err != nil {
		writeQuickPhraseErr(c, err)
		return
	}
	response.OK(c, nil)
}

func writeQuickPhraseErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrQuickPhraseNotFound):
		response.Fail(c, 404, 4004, err.Error())
	case errors.Is(err, service.ErrQuickPhraseEmpty),
		errors.Is(err, service.ErrQuickPhraseTooLong),
		errors.Is(err, service.ErrQuickPhraseTooMany):
		response.BadRequest(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}
