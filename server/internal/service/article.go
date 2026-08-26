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
	var a model.Article
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
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