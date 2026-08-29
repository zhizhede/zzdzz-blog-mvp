package service

import (
	"errors"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrArticleNotFound       = errors.New("article not found")
	ErrCategoryNotFoundArt   = errors.New("category not found")
)

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
	// IncludeAll: true 表示返回全部可见性(供 admin 后台); false 仅 public(供公开页)
	IncludeAll bool
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
	if !q.IncludeAll {
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

func (s *ArticleService) Get(id uint64) (*model.Article, error) {
	return s.getInternal(id, false)
}

// GetForAdmin 与 Get 等价,但能看到 private/draft。供 admin 后台编辑/预览用。
func (s *ArticleService) GetForAdmin(id uint64) (*model.Article, error) {
	return s.getInternal(id, true)
}

func (s *ArticleService) getInternal(id uint64, includeNonPublic bool) (*model.Article, error) {
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	// 公开访客读 private/draft → 当作不存在(404), 不暴露存在性
	if !includeNonPublic && a.Visibility != "public" {
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
	}
	if err := s.db.Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ArticleService) Update(id uint64, in ArticleInput) (*model.Article, error) {
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
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
	if err := s.db.Save(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *ArticleService) Delete(id uint64) error {
	res := s.db.Delete(&model.Article{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrArticleNotFound
	}
	return nil
}