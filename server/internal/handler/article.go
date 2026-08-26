package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

type ArticleHandler struct {
	svc *service.ArticleService
}

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

type articleReq struct {
	Title      string `json:"title" binding:"required,min=1,max=255"`
	Slug       string `json:"slug" binding:"max=255"`
	Summary    string `json:"summary" binding:"max=500"`
	Content    string `json:"content" binding:"required"`
	CategoryID uint64 `json:"category_id" binding:"required"`
}

func (h *ArticleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	catID, _ := strconv.ParseUint(c.Query("category_id"), 10, 64)
	keyword := c.Query("q")

	res, err := h.svc.List(service.ArticleListQuery{
		Page:       page,
		PageSize:   size,
		CategoryID: catID,
		Keyword:    keyword,
	})
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, res)
}

func (h *ArticleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	a, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			response.Fail(c, 404, 4004, "article not found")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, a)
}

func (h *ArticleHandler) Create(c *gin.Context) {
	var req articleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a, err := h.svc.Create(service.ArticleInput{
		Title:      req.Title,
		Slug:       req.Slug,
		Summary:    req.Summary,
		Content:    req.Content,
		CategoryID: req.CategoryID,
	})
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFoundArt) {
			response.Fail(c, 404, 4004, "category not found")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, a)
}

func (h *ArticleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req articleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a, err := h.svc.Update(id, service.ArticleInput{
		Title:      req.Title,
		Slug:       req.Slug,
		Summary:    req.Summary,
		Content:    req.Content,
		CategoryID: req.CategoryID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			response.Fail(c, 404, 4004, "article not found")
		case errors.Is(err, service.ErrCategoryNotFoundArt):
			response.Fail(c, 404, 4004, "category not found")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, a)
}

func (h *ArticleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			response.Fail(c, 404, 4004, "article not found")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, nil)
}