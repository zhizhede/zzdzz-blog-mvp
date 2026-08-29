package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/model"
	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

// mustBool 从 gin.Context 取 bool, 类型不匹配视为 false.
// 必须用 helper, 不能直接用 _, ok := c.Get(...) 当成 bool 用——
// c.Get 拿的是 any, 值是 false 时也会"存在", 必须断言类型.
func mustBool(c *gin.Context, key string) bool {
	v, ok := c.Get(key)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

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
	Visibility string `json:"visibility" binding:"omitempty,oneof=public private draft"`
}

func (h *ArticleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	catID, _ := strconv.ParseUint(c.Query("category_id"), 10, 64)
	keyword := c.Query("q")

	// 仅 admin 看到全部可见性; 普通登录用户与匿名一致, 只看 public
	_, includeAll := c.Get("is_admin")
	includeAll = includeAll && mustBool(c, "is_admin")

	res, err := h.svc.List(service.ArticleListQuery{
		Page:       page,
		PageSize:   size,
		CategoryID: catID,
		Keyword:    keyword,
		IncludeAll: includeAll,
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
	// 鉴权过且是 admin → 走 GetForAdmin, 可读 private/draft; 普通登录用户与匿名一致
	isAdmin := mustBool(c, "is_admin")
	var (
		a    *model.Article
		serr error
	)
	if isAdmin {
		a, serr = h.svc.GetForAdmin(id)
	} else {
		a, serr = h.svc.Get(id)
	}
	if serr != nil {
		if errors.Is(serr, service.ErrArticleNotFound) {
			response.Fail(c, 404, 4004, "article not found")
			return
		}
		response.ServerError(c, serr.Error())
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
		Visibility: req.Visibility,
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
		Visibility: req.Visibility,
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