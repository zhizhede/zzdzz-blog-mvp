package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

type CategoryHandler struct {
	svc *service.CategoryService
}

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type categoryReq struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
	Slug string `json:"slug" binding:"max=64"`
}

func (h *CategoryHandler) List(c *gin.Context) {
	cats, err := h.svc.List()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, cats)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "name required")
		return
	}
	cat, err := h.svc.Create(service.CategoryInput{Name: req.Name, Slug: req.Slug})
	if err != nil {
		if errors.Is(err, service.ErrCategoryNameConflict) {
			response.Fail(c, 409, 4009, "category name already exists")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req categoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "name required")
		return
	}
	cat, err := h.svc.Update(id, service.CategoryInput{Name: req.Name, Slug: req.Slug})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			response.Fail(c, 404, 4004, "category not found")
		case errors.Is(err, service.ErrCategoryNameConflict):
			response.Fail(c, 409, 4009, "category name already exists")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			response.Fail(c, 404, 4004, "category not found")
		case errors.Is(err, service.ErrCategoryHasArticles):
			response.Fail(c, 409, 4009, "category has articles, cannot delete")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
}