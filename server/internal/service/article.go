package service

import (
	"errors"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrArticleNotFound     = errors.New("article not found")
	ErrCategoryNotFoundArt = errors.New("category not found")
	ErrArticleNotOwned     = errors.New("not article owner")
)

// Actor 表示当前请求的调用者, 由 handler 从 context 注入.
// 暂时只关心是否管理员与 user id, 后续要加角色时扩字段即可.
type Actor struct {
	UserID  uint64
	IsAdmin bool
}

type ArticleService struct {
	db *gorm.DB
}

func NewArticleService(db *gorm.DB) *ArticleService {
	return &ArticleService{db: db}
}

type ArticleListQuery struct {
	Page       int
	PageSize   int
	CategoryID uint64
	Keyword    string
	// IncludeAll: true 表示返回全部可见性(供 admin 后台); false 仅 public(供公开页/非 admin 登录用户)
	IncludeAll bool
	// AuthorID: 非空时按作者过滤, 同时强制返回该作者的全部可见性(自己的文章自己全看)
	AuthorID *uint64
}

type ArticleListResult struct {
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	Size    int             `json:"size"`
	Items   []model.Article `json:"items"`
}

func (s *ArticleService) List(q ArticleListQuery) (*ArticleListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 10
	}

	tx := s.db.Model(&model.Article{})
	if q.CategoryID > 0 {
		tx = tx.Where("category_id = ?", q.CategoryID)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("title ILIKE ? OR summary ILIKE ?", kw, kw)
	}
	if q.AuthorID != nil {
		// 指定作者时, 返回该作者的全部可见性(自己看自己不需要被 visibility 限制)
		tx = tx.Where("author_id = ?", *q.AuthorID)
	} else if !q.IncludeAll {
		tx = tx.Where("visibility = ?", "public")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []model.Article
	if err := tx.Order("created_at DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &ArticleListResult{
		Total: total,
		Page:  q.Page,
		Size:  q.PageSize,
		Items: items,
	}, nil
}

// Get 公开访客或非作者读 — 仅返回 public 文章, 其它 404(不暴露存在性)
func (s *ArticleService) Get(id uint64) (*model.Article, error) {
	return s.getInternal(id, false, nil)
}

// GetForAdmin admin 后台读 — 返回所有可见性
func (s *ArticleService) GetForAdmin(id uint64) (*model.Article, error) {
	return s.getInternal(id, true, nil)
}

// GetForOwner 文章作者读自己的文章 — 返回所有可见性(包括 private/draft)
func (s *ArticleService) GetForOwner(id uint64, ownerID uint64) (*model.Article, error) {
	return s.getInternal(id, true, &ownerID)
}

func (s *ArticleService) getInternal(id uint64, includeNonPublic bool, owner *uint64) (*model.Article, error) {
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	// 作者自己: 全部可见性都返回
	if owner != nil && a.AuthorID != nil && *a.AuthorID == *owner {
		// 走 includeNonPublic=true 的逻辑(已经放行)
	} else if !includeNonPublic && a.Visibility != "public" {
		// 公开访客读 private/draft -> 当作不存在(404), 不暴露存在性
		return nil, ErrArticleNotFound
	}
	s.db.Model(&a).UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	a.ViewCount++
	return &a, nil
}

type ArticleInput struct {
	Title      string
	Slug       string
	Summary    string
	Content    string
	CategoryID uint64
	Visibility string // 默认 "public"
	// AuthorID 由 handler 从 context 注入, 不来自请求体, 防止越权伪造作者
	AuthorID *uint64
}

// normalizeVisibility: 空 -> public; 非 public/private/draft -> public(向后兼容)
func normalizeVisibility(v string) string {
	switch v {
	case "public", "private", "draft":
		return v
	default:
		return "public"
	}
}

func (s *ArticleService) Create(in ArticleInput) (*model.Article, error) {
	var cat model.Category
	if err := s.db.First(&cat, in.CategoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFoundArt
		}
		return nil, err
	}
	a := &model.Article{
		Title:      in.Title,
		Slug:       in.Slug,
		Summary:    in.Summary,
		Content:    in.Content,
		CategoryID: in.CategoryID,
		Visibility: normalizeVisibility(in.Visibility),
		AuthorID:   in.AuthorID,
	}
	if err := s.db.Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ArticleService) Update(id uint64, in ArticleInput, actor Actor) (*model.Article, error) {
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	if !canEdit(a, actor) {
		return nil, ErrArticleNotOwned
	}
	if in.CategoryID != a.CategoryID {
		var cat model.Category
		if err := s.db.First(&cat, in.CategoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCategoryNotFoundArt
			}
			return nil, err
		}
	}
	a.Title = in.Title
	a.Slug = in.Slug
	a.Summary = in.Summary
	a.Content = in.Content
	a.CategoryID = in.CategoryID
	a.Visibility = normalizeVisibility(in.Visibility)
	// 不允许通过 Update 接口改 author_id; Create 时由 handler 注入, 后续要"转让"再加专门接口
	if err := s.db.Save(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *ArticleService) Delete(id uint64, actor Actor) error {
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	if !canEdit(a, actor) {
		return ErrArticleNotOwned
	}
	res := s.db.Delete(&model.Article{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// canEdit 权限规则:
//   - admin: 全部可编辑(含 author_id=NULL 的老文章)
//   - 非 admin: 只能编辑自己作者的文章(author_id == actor.UserID)
func canEdit(a model.Article, actor Actor) bool {
	if actor.IsAdmin {
		return true
	}
	if a.AuthorID == nil {
		return false
	}
	return *a.AuthorID == actor.UserID
}