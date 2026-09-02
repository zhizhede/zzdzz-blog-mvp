package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
)

// vectorDims 与 migrations/0009 的 vector 列维度一致, 换模型/维度需新迁移 + 全量重嵌.
// 实测 MiniMax embo-01 返回 1536 维(官方文档标注 1024 与实际不符).
const vectorDims = 1536

const sourceTypeArticle = "article"

// AIChunk 向量索引行. embedding 列不映射进结构体, 插入/检索都走 raw SQL +
// pgvector 字面量(见 embedding.go vectorLiteral).
type AIChunk struct {
	ID         uint64  `gorm:"primaryKey"`
	SourceType string  `gorm:"size:16;not null;default:article"`
	SourceID   uint64  `gorm:"not null"`
	OwnerID    *uint64 // 冗余 articles.author_id, NULL = 老文章
	Scope      string  `gorm:"size:16;not null"` // public / private
	ChunkIndex int     `gorm:"not null"`
	Heading    *string
	Content    string
	Metadata   []byte // JSONB
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (AIChunk) TableName() string { return "ai_chunks" }

// chunkMeta 引用展示与向量语义都需要的文章级元数据, 存 JSONB.
type chunkMeta struct {
	Title       string   `json:"title"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PublishedAt string   `json:"published_at"` // 2006-01-02, 时间语义必须进上下文
}

// AccessQuery 检索判权入参. 未来开放匿名"问博客"时加 Anonymous 字段,
// 只改 scopeFilter 一处, 见 doc/v0.3-tech-design.md §8.
type AccessQuery struct {
	UserID  uint64
	IsAdmin bool
}

// RecallHit 命中的一块内容.
type RecallHit struct {
	ArticleID uint64
	Heading   string
	Content   string
	Scope     string // public / private, 注入 prompt 时标注"公开文章/私人笔记"
	Meta      chunkMeta
	Score     float64
}

// RecallSource 推给前端的引用来源(SSE sources 事件), 字段即 JSON 契约.
type RecallSource struct {
	Index     int     `json:"index"`
	ArticleID uint64  `json:"article_id"`
	Title     string  `json:"title"`
	Heading   string  `json:"heading,omitempty"`
	Scope     string  `json:"scope"`
	Score     float64 `json:"score"`
}

// RecallService AI 回顾: 把用户写过的内容向量化, 对话时按可见性过滤召回.
// embed 为 nil 时整个服务处于禁用态(Enabled=false), 所有入口静默返回.
type RecallService struct {
	db           *gorm.DB
	embed        EmbeddingClient
	topK         int
	minScore     float64
	indexPrivate bool
}

func NewRecallService(db *gorm.DB, embed EmbeddingClient, cfg config.RecallConfig) *RecallService {
	s := &RecallService{
		db:           db,
		embed:        embed,
		topK:         cfg.TopK,
		minScore:     cfg.MinScore,
		indexPrivate: cfg.IndexPrivate,
	}
	if s.topK <= 0 {
		s.topK = 5
	}
	if s.minScore <= 0 {
		s.minScore = 0.45
	}
	return s
}

// Enabled 用 nil 接收者判断, handler/service 侧不需要判空.
func (s *RecallService) Enabled() bool { return s != nil && s.embed != nil }

// -------------------- 索引 --------------------

// IndexArticle 全量重建一篇文章的向量块: 先嵌入, 成功后事务内删旧插新, 幂等可重跑.
// draft 不入索引; index_private=false 时私人文章清块不入索引.
func (s *RecallService) IndexArticle(ctx context.Context, a *model.Article) error {
	if !s.Enabled() {
		return nil
	}
	if a.Visibility == "draft" || (a.Visibility == "private" && !s.indexPrivate) {
		return s.RemoveArticle(a.ID)
	}

	meta := s.articleMetadata(ctx, a)
	chunks := SplitMarkdown(a.Content)
	if len(chunks) == 0 {
		return s.RemoveArticle(a.ID)
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = embedText(a, meta, c)
	}
	vecs, err := s.embed.Embed(ctx, texts)
	if err != nil {
		return err
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"DELETE FROM ai_chunks WHERE source_type = ? AND source_id = ?",
			sourceTypeArticle, a.ID,
		).Error; err != nil {
			return err
		}
		for i, c := range chunks {
			heading := c.Heading
			if err := tx.Exec(
				`INSERT INTO ai_chunks
				 (source_type, source_id, owner_id, scope, chunk_index, heading, content, metadata, embedding)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?::vector)`,
				sourceTypeArticle, a.ID, a.AuthorID, a.Visibility, i, heading, c.Content, metaJSON, vectorLiteral(vecs[i]),
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveArticle 删除一篇文章的全部向量块(文章删除 / 退回草稿 / 关闭私人索引时调).
func (s *RecallService) RemoveArticle(sourceID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.db.Exec(
		"DELETE FROM ai_chunks WHERE source_type = ? AND source_id = ?",
		sourceTypeArticle, sourceID,
	).Error
}

// UpdateScope 只改过滤列, 不重新嵌入——纯 public<->private 翻转的轻量路径.
func (s *RecallService) UpdateScope(sourceID uint64, scope string) error {
	if !s.Enabled() {
		return nil
	}
	if scope == "draft" || (scope == "private" && !s.indexPrivate) {
		return s.RemoveArticle(sourceID)
	}
	return s.db.Model(&AIChunk{}).
		Where("source_type = ? AND source_id = ?", sourceTypeArticle, sourceID).
		Update("scope", scope).Error
}

// OnVisibilityChanged 可见性变化后的索引维护, 策略:
//   - 变 draft: 删块(草稿是高频自动保存数据, 不进索引)
//   - 已有块(纯可见性翻转): 只改 scope 列, 不重嵌
//   - 首次从 draft 出来(无块): 全量建索引
func (s *RecallService) OnVisibilityChanged(ctx context.Context, a *model.Article) error {
	if !s.Enabled() {
		return nil
	}
	if a.Visibility == "draft" {
		return s.RemoveArticle(a.ID)
	}
	var n int64
	if err := s.db.Model(&AIChunk{}).
		Where("source_type = ? AND source_id = ?", sourceTypeArticle, a.ID).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return s.UpdateScope(a.ID, a.Visibility)
	}
	return s.IndexArticle(ctx, a)
}

// articleMetadata 拉分类名与标签名, 失败不致命(缺元数据只影响语义质量).
func (s *RecallService) articleMetadata(ctx context.Context, a *model.Article) chunkMeta {
	meta := chunkMeta{Title: a.Title}
	if a.CreatedAt.IsZero() {
		meta.PublishedAt = ""
	} else {
		meta.PublishedAt = a.CreatedAt.Format("2006-01-02")
	}
	var cat model.Category
	if err := s.db.WithContext(ctx).First(&cat, a.CategoryID).Error; err == nil {
		meta.Category = cat.Name
	}
	var tags []model.Tag
	if err := s.db.WithContext(ctx).Raw(`
		SELECT t.* FROM tags t
		JOIN article_tags at ON at.tag_id = t.id
		WHERE at.article_id = ?
		ORDER BY t.id ASC
	`, a.ID).Scan(&tags).Error; err == nil {
		for _, t := range tags {
			meta.Tags = append(meta.Tags, t.Name)
		}
	}
	return meta
}

// embedText 拼进标题/时间/分类/标签, 让"什么时候写的、关于什么"进入向量语义.
// "回顾以前发的东西"场景里用户会问时间性问题, 只有正文向量答不出来.
func embedText(a *model.Article, m chunkMeta, c Chunk) string {
	var b strings.Builder
	fmt.Fprintf(&b, "《%s》", a.Title)
	if m.PublishedAt != "" {
		fmt.Fprintf(&b, " · %s", m.PublishedAt)
	}
	if m.Category != "" {
		fmt.Fprintf(&b, " · %s", m.Category)
	}
	if len(m.Tags) > 0 {
		fmt.Fprintf(&b, " · #%s", strings.Join(m.Tags, " #"))
	}
	b.WriteString("\n")
	if c.Heading != "" {
		b.WriteString("## " + c.Heading + "\n")
	}
	b.WriteString(c.Content)
	return b.String()
}

// -------------------- 检索 --------------------

// RecallForQuery 对一条用户消息做检索, 返回注入用的 system prompt 与引用来源.
// 无命中(或全低于阈值)时返回空, 调用方按普通对话继续.
func (s *RecallService) RecallForQuery(ctx context.Context, q AccessQuery, userText string) (string, []RecallSource, error) {
	hits, err := s.Search(ctx, q, userText)
	if err != nil {
		return "", nil, err
	}
	if len(hits) == 0 {
		return "", nil, nil
	}
	sources := make([]RecallSource, len(hits))
	var b strings.Builder
	b.WriteString("下面是从该用户可见的历史文章中检索到的相关片段,每段出处行标注了编号、标题、发布时间和可见范围。")
	b.WriteString("回答时可以自然地引用这些内容(例如\"你在 2025 年 3 月写过……\"),但不得虚构片段里没有的内容;与问题无关的片段请忽略。\n\n")
	for i, h := range hits {
		sources[i] = RecallSource{
			Index:     i + 1,
			ArticleID: h.ArticleID,
			Title:     h.Meta.Title,
			Heading:   h.Heading,
			Scope:     h.Scope,
			Score:     h.Score,
		}
		scopeName := "公开文章"
		if h.Scope == "private" {
			scopeName = "私人笔记"
		}
		fmt.Fprintf(&b, "[%d] 《%s》(%s, %s)\n%s\n\n", i+1, h.Meta.Title, h.Meta.PublishedAt, scopeName, h.Content)
	}
	return b.String(), sources, nil
}

// Search 向量检索, 判权规则与 ArticleService 一致:
//   - admin: 全部已索引内容
//   - 普通用户: public + 自己的 private; 老文章(owner 为 NULL)的 private 仅 admin
//
// 未来开放匿名读者只查 public 时, 改这里一个函数即可.
func (s *RecallService) Search(ctx context.Context, q AccessQuery, query string) ([]RecallHit, error) {
	if !s.Enabled() || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	// 查询方向嵌入: 非对称模型(embo-01)用 type=query, 与入库的 type=db 对应
	vec, err := s.embed.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	lit := vectorLiteral(vec)

	var rows []struct {
		SourceID uint64  `gorm:"column:source_id"`
		Heading  *string `gorm:"column:heading"`
		Content  string  `gorm:"column:content"`
		Scope    string  `gorm:"column:scope"`
		Metadata []byte  `gorm:"column:metadata"`
		Score    float64 `gorm:"column:score"`
	}
	err = s.db.WithContext(ctx).Raw(`
		SELECT source_id, heading, content, scope, metadata,
		       1 - (embedding <=> ?::vector) AS score
		FROM ai_chunks
		WHERE scope = 'public'
		   OR (scope = 'private' AND owner_id = ?)
		   OR (scope = 'private' AND owner_id IS NULL AND ?)
		ORDER BY embedding <=> ?::vector
		LIMIT ?
	`, lit, q.UserID, q.IsAdmin, lit, s.topK).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	hits := make([]RecallHit, 0, len(rows))
	for _, r := range rows {
		if r.Score < s.minScore {
			continue
		}
		h := RecallHit{
			ArticleID: r.SourceID,
			Content:   r.Content,
			Scope:     r.Scope,
			Score:     r.Score,
		}
		if r.Heading != nil {
			h.Heading = *r.Heading
		}
		if len(r.Metadata) > 0 {
			_ = json.Unmarshal(r.Metadata, &h.Meta)
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// -------------------- 兜底清理(配置翻转时) --------------------

// PruneByScope 按 index_private 配置清理不再允许的块. 配置从 true 翻 false 后
// 调一次即可, 目前在进程启动时执行.
func (s *RecallService) PruneByScope() error {
	if !s.Enabled() || s.indexPrivate {
		return nil
	}
	return s.db.Exec("DELETE FROM ai_chunks WHERE scope = ?", "private").Error
}
