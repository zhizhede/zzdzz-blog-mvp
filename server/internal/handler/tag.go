package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

type TagHandler struct {
	svc *service.TagService
}

func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

type tagReq struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
	Slug string `json:"slug" binding:"max=64"`
}

func (h *TagHandler) List(c *gin.Context) {
	tags, err := h.svc.List()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, tags)
}

func (h *TagHandler) Create(c *gin.Context) {
	var req tagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	t, err := h.svc.Create(req.Name, req.Slug)
	if err != nil {
		if errors.Is(err, service.ErrTagNameTaken) {
			response.Fail(c, 409, 4009, "标签名或别名已存在")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, t)
}

func (h *TagHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req tagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	t, err := h.svc.Update(id, req.Name, req.Slug)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			response.Fail(c, 404, 4004, "标签不存在")
		case errors.Is(err, service.ErrTagNameTaken):
			response.Fail(c, 409, 4009, "标签名或别名已存在")
		default:
			response.BadRequest(c, err.Error())
		}
		return
	}
	response.OK(c, t)
}

func (h *TagHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrTagNotFound) {
			response.Fail(c, 404, 4004, "标签不存在")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, nil)
}
