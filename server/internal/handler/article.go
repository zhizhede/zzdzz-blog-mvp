package handler

import (
	"errors"
	"strconv"
	"time"

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

// actorOf 从 context 构造 Actor. 匿名访问时 user_id 为 0, is_admin 为 false.
func actorOf(c *gin.Context) service.Actor {
	return service.Actor{
		UserID:  userIDOf(c),
		IsAdmin: mustBool(c, "is_admin"),
	}
}

type ArticleHandler struct {
	svc *service.ArticleService
}

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

type articleReq struct {
	Title      string   `json:"title" binding:"required,min=1,max=255"`
	Slug       string   `json:"slug" binding:"max=255"`
	Summary    string   `json:"summary" binding:"max=500"`
	Content    string   `json:"content" binding:"required"`
	CategoryID uint64   `json:"category_id" binding:"required"`
	Visibility string   `json:"visibility" binding:"omitempty,oneof=public private draft"`
	// TagIDs 可选; 不传不动标签, 传 [] 表示清空, 传 [1,2,3] 表示 replace
	TagIDs *[]uint64 `json:"tag_ids" binding:"omitempty"`
}

type autosaveReq struct {
	Title      string `json:"title" binding:"required,min=1,max=255"`
	Summary    string `json:"summary" binding:"max=500"`
	Content    string `json:"content" binding:"required"`
	CategoryID uint64 `json:"category_id"`
}

func (h *ArticleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	catID, _ := strconv.ParseUint(c.Query("category_id"), 10, 64)
	keyword := c.Query("q")
	authorID, _ := strconv.ParseUint(c.Query("author_id"), 10, 64)
	tagID, _ := strconv.ParseUint(c.Query("tag_id"), 10, 64)

	isAdmin := mustBool(c, "is_admin")
	q := service.ArticleListQuery{
		Page:       page,
		PageSize:   size,
		CategoryID: catID,
		Keyword:    keyword,
		IncludeAll: isAdmin,
		TagID:      tagID,
	}

	// 优先级 1: 显式传 author_id(= 自己), 用于"我的笔记"; 必须登录, 否则越权
	if uid := userIDOf(c); uid > 0 && authorID == uid {
		q.AuthorID = &uid
		q.IncludeAll = false // 由 AuthorID 分支接管可见性过滤
	}

	res, err := h.svc.List(q)
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
	isAdmin := mustBool(c, "is_admin")
	uid := userIDOf(c)
	var (
		a    *model.Article
		serr error
	)
	switch {
	case isAdmin:
		a, serr = h.svc.GetForAdmin(id)
	case uid > 0:
		// 登录非 admin: 走 GetForOwner, 让作者能看到自己的 private/draft
		a, serr = h.svc.GetForOwner(id, uid)
	default:
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
	in := service.ArticleInput{
		Title:      req.Title,
		Slug:       req.Slug,
		Summary:    req.Summary,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		Visibility: req.Visibility,
		TagIDs:     req.TagIDs,
	}
	// 由 handler 从 context 注入 author_id, 防止请求体伪造作者
	if uid := userIDOf(c); uid > 0 {
		in.AuthorID = &uid
	}
	a, err := h.svc.Create(in)
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
		TagIDs:     req.TagIDs,
		// Update 不允许改 author_id, service 忽略此字段
	}, actorOf(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			response.Fail(c, 404, 4004, "article not found")
		case errors.Is(err, service.ErrCategoryNotFoundArt):
			response.Fail(c, 404, 4004, "category not found")
		case errors.Is(err, service.ErrArticleNotOwned):
			response.Fail(c, 403, 4003, "not article owner")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, a)
}

// Autosave 草稿自动保存, 只接白名单字段, 只允许 draft 文章, 写 last_autosaved_at 不动 updated_at.
func (h *ArticleHandler) Autosave(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req autosaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a, err := h.svc.Autosave(id, service.AutosaveInput{
		Title:      req.Title,
		Summary:    req.Summary,
		Content:    req.Content,
		CategoryID: req.CategoryID,
	}, actorOf(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			response.Fail(c, 404, 4004, "article not found")
		case errors.Is(err, service.ErrArticleNotOwned):
			response.Fail(c, 403, 4003, "not article owner")
		case errors.Is(err, service.ErrArticleNotDraft):
			response.Fail(c, 400, 4002, "only drafts can be autosaved")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, gin.H{
		"id":                 a.ID,
		"last_autosaved_at":  a.LastAutosavedAt,
		"server_received_at": time.Now().UTC(),
	})
}

// ListMyDrafts 列出当前登录用户的 draft, 按 last_autosaved_at DESC.
func (h *ArticleHandler) ListMyDrafts(c *gin.Context) {
	uid := userIDOf(c)
	if uid == 0 {
		response.Fail(c, 401, 4001, "login required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	items, total, err := h.svc.ListMyDrafts(uid, page, size)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"items": items,
	})
}

// GetWithTags 返回单篇文章 + 标签, 给详情/编辑用.
func (h *ArticleHandler) GetWithTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	isAdmin := mustBool(c, "is_admin")
	uid := userIDOf(c)
	var (
		res *service.ArticleWithTags
		serr error
	)
	switch {
	case isAdmin:
		full, err := h.svc.GetForAdmin(id)
		if err != nil {
			serr = err
			break
		}
		tags, _ := h.svc.TagsForArticle(id)
		res = &service.ArticleWithTags{Article: *full, Tags: tags}
	case uid > 0:
		full, err := h.svc.GetForOwner(id, uid)
		if err != nil {
			serr = err
			break
		}
		tags, _ := h.svc.TagsForArticle(id)
		res = &service.ArticleWithTags{Article: *full, Tags: tags}
	default:
		full, err := h.svc.Get(id)
		if err != nil {
			serr = err
			break
		}
		tags, _ := h.svc.TagsForArticle(id)
		res = &service.ArticleWithTags{Article: *full, Tags: tags}
	}
	if serr != nil {
		if errors.Is(serr, service.ErrArticleNotFound) {
			response.Fail(c, 404, 4004, "article not found")
			return
		}
		response.ServerError(c, serr.Error())
		return
	}
	response.OK(c, res)
}

func (h *ArticleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(id, actorOf(c)); err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			response.Fail(c, 404, 4004, "article not found")
		case errors.Is(err, service.ErrArticleNotOwned):
			response.Fail(c, 403, 4003, "not article owner")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
}