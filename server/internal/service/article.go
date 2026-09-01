package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrArticleNotFound     = errors.New("article not found")
	ErrCategoryNotFoundArt = errors.New("category not found")
	ErrArticleNotOwned     = errors.New("not article owner")
	ErrArticleNotDraft     = errors.New("article is not a draft, use Update to modify")
	ErrTagNotFound         = errors.New("tag not found")
	ErrTagNameTaken        = errors.New("tag name already exists")
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
	// TagID: 非 0 时按标签过滤, 仅返回 public + 含此标签的文章
	TagID uint64
	// OnlyDrafts: true 时仅返回 visibility='draft' 的文章, 与 AuthorID 配合实现「我的草稿列表」
	OnlyDrafts bool
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
	if q.TagID > 0 {
		// 反向 JOIN article_tags, 单标签过滤
		tx = tx.Joins("JOIN article_tags at ON at.article_id = articles.id").
			Where("at.tag_id = ?", q.TagID)
	}
	if q.OnlyDrafts {
		tx = tx.Where("visibility = ?", "draft")
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
	// TagIDs 可选; 为 nil/空时不动标签, 非 nil 时服务端 replace 该文章的全部标签关联
	TagIDs *[]uint64
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
	if in.TagIDs != nil {
		if err := s.replaceArticleTags(a.ID, *in.TagIDs); err != nil {
			return nil, err
		}
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
	if in.TagIDs != nil {
		if err := s.replaceArticleTags(a.ID, *in.TagIDs); err != nil {
			return nil, err
		}
	}
	return &a, nil
}

// SetVisibility 单独改可见性: 不管文章当前是 public/private/draft 哪种状态,
// 只要是作者本人或 admin, 任何时候都能改. 只动 visibility 一列, 不碰正文/分类/标签.
func (s *ArticleService) SetVisibility(id uint64, visibility string, actor Actor) (*model.Article, error) {
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
	a.Visibility = normalizeVisibility(visibility)
	if err := s.db.Model(&a).UpdateColumn("visibility", a.Visibility).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// replaceArticleTags 用事务把 article_id 关联的 tag 全部替换为 tagIDs.
// 空切片表示清空(取消所有标签).
func (s *ArticleService) replaceArticleTags(articleID uint64, tagIDs []uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM article_tags WHERE article_id = ?", articleID).Error; err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}
		// 仅保留 tagIDs 中实际存在的, 防止客户端瞎传
		var existing []model.Tag
		if err := tx.Where("id IN ?", tagIDs).Find(&existing).Error; err != nil {
			return err
		}
		if len(existing) == 0 {
			return nil
		}
		rows := make([]map[string]any, 0, len(existing))
		for _, t := range existing {
			rows = append(rows, map[string]any{
				"article_id": articleID,
				"tag_id":     t.ID,
			})
		}
		return tx.Table("article_tags").Create(rows).Error
	})
}

// TagsForArticle 拉取一篇文章关联的所有 tag (按 id 升序, 稳定).
func (s *ArticleService) TagsForArticle(articleID uint64) ([]model.Tag, error) {
	var tags []model.Tag
	err := s.db.Raw(`
		SELECT t.* FROM tags t
		JOIN article_tags at ON at.tag_id = t.id
		WHERE at.article_id = ?
		ORDER BY t.id ASC
	`, articleID).Scan(&tags).Error
	return tags, err
}

// ArticleWithTags 把 tags 内联进 article JSON, 便于单次返回给前端.
type ArticleWithTags struct {
	model.Article
	Tags []model.Tag `json:"tags"`
}

// GetWithTags 返回文章 + 其标签列表. 用于详情/编辑.
func (s *ArticleService) GetWithTags(id uint64) (*ArticleWithTags, error) {
	a, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	tags, err := s.TagsForArticle(id)
	if err != nil {
		return nil, err
	}
	return &ArticleWithTags{Article: *a, Tags: tags}, nil
}

// ListItemsWithTags 返回带 tags 的 items, 走与 List 一致的过滤, 但每条 item 含 tags.
// 前端如果不需要 tags, 用 List 更便宜.
func (s *ArticleService) ListItemsWithTags(q ArticleListQuery) ([]ArticleWithTags, error) {
	base, err := s.List(q)
	if err != nil {
		return nil, err
	}
	if len(base.Items) == 0 {
		return []ArticleWithTags{}, nil
	}
	ids := make([]uint64, len(base.Items))
	for i, a := range base.Items {
		ids[i] = a.ID
	}
	var rows []struct {
		ArticleID uint64 `gorm:"column:article_id"`
		model.Tag
	}
	if err := s.db.Raw(`
		SELECT at.article_id, t.* FROM tags t
		JOIN article_tags at ON at.tag_id = t.id
		WHERE at.article_id IN ? ORDER BY t.id ASC
	`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	bucket := make(map[uint64][]model.Tag, len(ids))
	for _, r := range rows {
		bucket[r.ArticleID] = append(bucket[r.ArticleID], r.Tag)
	}
	out := make([]ArticleWithTags, len(base.Items))
	for i, a := range base.Items {
		out[i] = ArticleWithTags{Article: a, Tags: bucket[a.ID]}
	}
	return out, nil
}

// AutosaveInput 只接白名单字段, 不改 visibility / slug / view_count / author_id.
type AutosaveInput struct {
	Title      string
	Summary    string
	Content    string
	CategoryID uint64
}

// Autosave 自动保存:
//   - 强制 visibility='draft' (新建时由 Create 给 draft, 已发布的不能再被 autosave 接口写)
//   - 只在 actor 是作者本人 / admin 时允许
//   - 写 last_autosaved_at, 不动 updated_at, 不影响列表排序
//   - 自动塞 author_id 为空时不动 (已有文章保留原 author)
func (s *ArticleService) Autosave(id uint64, in AutosaveInput, actor Actor) (*model.Article, error) {
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
	// autosave 不允许覆盖"已发布"的文章, 避免误操作把 public 变 draft
	// 实际上, 草稿自动保存只针对 draft; 其他可见性按 Update 走正常接口
	if a.Visibility != "draft" {
		return nil, ErrArticleNotDraft
	}
	if in.CategoryID != 0 && in.CategoryID != a.CategoryID {
		var cat model.Category
		if err := s.db.First(&cat, in.CategoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCategoryNotFoundArt
			}
			return nil, err
		}
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"title":             in.Title,
		"summary":           in.Summary,
		"content":           in.Content,
		"last_autosaved_at": now,
	}
	if in.CategoryID != 0 {
		updates["category_id"] = in.CategoryID
	}
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}
	a.LastAutosavedAt = &now
	return &a, nil
}

// ListMyDrafts 列出某个用户自己的 draft, 按 last_autosaved_at DESC (NULL 排到最后).
func (s *ArticleService) ListMyDrafts(userID uint64, page, size int) ([]model.Article, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	q := s.db.Model(&model.Article{}).
		Where("author_id = ? AND visibility = ?", userID, "draft")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Article
	if err := q.Order("last_autosaved_at DESC NULLS LAST, id DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
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