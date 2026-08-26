package service

import (
	"errors"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryHasArticles  = errors.New("category has articles, cannot delete")
	ErrCategoryNameConflict = errors.New("category name already exists")
)

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

func (s *CategoryService) List() ([]model.Category, error) {
	var cats []model.Category
	if err := s.db.Order("id ASC").Find(&cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

type CategoryInput struct {
	Name string
	Slug string
}

func (s *CategoryService) Create(in CategoryInput) (*model.Category, error) {
	c := &model.Category{Name: in.Name, Slug: in.Slug}
	if err := s.db.Create(c).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCategoryNameConflict
		}
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Update(id uint64, in CategoryInput) (*model.Category, error) {
	var c model.Category
	if err := s.db.First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	c.Name = in.Name
	c.Slug = in.Slug
	if err := s.db.Save(&c).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCategoryNameConflict
		}
		return nil, err
	}
	return &c, nil
}

func (s *CategoryService) Delete(id uint64) error {
	var count int64
	if err := s.db.Model(&model.Article{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryHasArticles
	}
	res := s.db.Delete(&model.Category{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (containsAny(err.Error(), "duplicate key", "unique constraint") ||
		containsAny(err.Error(), "unique_violation", "23505"))
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}