package service

import (
	"errors"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

type TagService struct {
	db *gorm.DB
}

func NewTagService(db *gorm.DB) *TagService {
	return &TagService{db: db}
}

// TagWithCount 给前端首页云用, 带「该标签下 public 文章数」
type TagWithCount struct {
	model.Tag
	Count int64 `json:"count"`
}

// List 返回全部标签 + 每个标签的公开文章数
func (s *TagService) List() ([]TagWithCount, error) {
	rows := []TagWithCount{} // 显式非 nil, 避免 GORM Scan 空表后变 nil 让 JSON 序列化成 null
	err := s.db.Raw(`
		SELECT t.*, COALESCE(c.cnt, 0) AS count
		FROM tags t
		LEFT JOIN (
			SELECT at.tag_id AS tag_id, COUNT(*) AS cnt
			FROM article_tags at
			JOIN articles a ON a.id = at.article_id
			WHERE a.visibility = 'public'
			GROUP BY at.tag_id
		) c ON c.tag_id = t.id
		ORDER BY count DESC, t.id ASC
	`).Scan(&rows).Error
	return rows, err
}

// Create 新建标签, name 必填且唯一, slug 可空 (空时从 name 派生)
func (s *TagService) Create(name, slug string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name required")
	}
	if len(name) > 64 {
		return nil, errors.New("name too long")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, errors.New("invalid name, slug cannot be derived")
	}
	// 重复检查
	var dup model.Tag
	if err := s.db.Where("name = ? OR slug = ?", name, slug).First(&dup).Error; err == nil {
		return nil, ErrTagNameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	t := &model.Tag{Name: name, Slug: slug}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

// Update 改名, 不动关联
func (s *TagService) Update(id uint64, name, slug string) (*model.Tag, error) {
	var t model.Tag
	if err := s.db.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name required")
	}
	if slug = strings.TrimSpace(slug); slug == "" {
		slug = slugify(name)
	}
	var dup model.Tag
	if err := s.db.Where("(name = ? OR slug = ?) AND id <> ?", name, slug, id).First(&dup).Error; err == nil {
		return nil, ErrTagNameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	t.Name = name
	t.Slug = slug
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete 删标签, article_tags 由 FK CASCADE 自动清
func (s *TagService) Delete(id uint64) error {
	res := s.db.Delete(&model.Tag{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}

// slugify 把 "Vue 3" / "Open AI" 变成 "vue-3" / "open-ai"
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '/':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := b.String()
	out = strings.TrimRight(out, "-")
	return out
}
