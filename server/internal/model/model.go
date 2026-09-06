package model

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// DeletedAt 软删除(0012 起): db.Delete 变标记删除, 查询自动过滤已删行.
	// 不序列化到 JSON——对外接口永远看不到已删数据.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	Base
	Username     string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"size:128;not null" json:"-"`
	IsActive     bool   `gorm:"not null;default:true" json:"is_active"`
	// IsAdmin 标记超级管理员. 当前由 service.Login 根据 ZZDZZ_ADMIN_USERNAMES 推断回填,
	// 数据库 schema 暂未落列; gorm 默认值 false, 真正取值以 service 为准.
	IsAdmin bool `gorm:"-" json:"is_admin"`
}

func (User) TableName() string { return "users" }

type Category struct {
	Base
	Name string `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Slug string `gorm:"size:64;index" json:"slug"`
}

func (Category) TableName() string { return "categories" }

type Article struct {
	Base
	Title      string `gorm:"size:255;not null;index" json:"title"`
	Slug       string `gorm:"size:255;index" json:"slug"`
	Summary    string `gorm:"size:500" json:"summary"`
	Content    string `gorm:"type:text;not null" json:"content"`
	CategoryID uint64 `gorm:"index;not null" json:"category_id"`
	ViewCount  int    `gorm:"default:0" json:"view_count"`
	// Visibility: public(默认) / private(仅自己) / draft(草稿, 连 URL 都404)
	Visibility string `gorm:"size:16;not null;default:public" json:"visibility"`
	// AuthorID 文章作者. 老文章保持 NULL, 仅 admin 可改; 非 admin 只能改自己作者的文章.
	// 指针类型使 JSON 输出 null 而非 0, 与"无作者"语义一致.
	AuthorID *uint64 `gorm:"index" json:"author_id"`
	// LastAutosavedAt 只在 autosave 接口里被更新, 列表排序仍按 updated_at,
	// 避免「每 8 秒写一次数据库」污染文章的"最后改动时间"语义.
	LastAutosavedAt *time.Time `json:"last_autosaved_at"`
}

func (Article) TableName() string { return "articles" }

type Tag struct {
	Base
	Name string `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Slug string `gorm:"size:64;uniqueIndex"        json:"slug"`
}

func (Tag) TableName() string { return "tags" }

type AIConversation struct {
	Base
	UserID uint64 `gorm:"index;not null" json:"user_id"`
	Title  string `gorm:"size:100;not null;default:'未命名会话'" json:"title"`
}

func (AIConversation) TableName() string { return "ai_conversations" }

type AIMessage struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	ConversationID uint64         `gorm:"index;not null" json:"conversation_id"`
	Role           string         `gorm:"size:16;not null" json:"role"`
	Content        string         `gorm:"type:text;not null;default:''" json:"content"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AIMessage) TableName() string { return "ai_messages" }

// StyleProfile 用户写作风格卡(AI 写作工作流, doc/v0.4-tech-design.md §5).
// 每用户一行; Source: auto(LLM 提炼) / manual(用户手改).
type StyleProfile struct {
	UserID    uint64         `gorm:"primaryKey;column:user_id" json:"user_id"`
	Profile   string         `gorm:"type:text;not null" json:"profile"`
	Source    string         `gorm:"size:16;not null;default:auto" json:"source"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (StyleProfile) TableName() string { return "style_profiles" }

// ArticleVersion 文章版本快照(AI 采用前 / 回滚前自动写入, doc/v0.4-tech-design.md §6).
// 只增不改: 回滚也先把当前内容存一条 pre_restore 再写回.
type ArticleVersion struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	ArticleID uint64 `gorm:"not null" json:"article_id"`
	Title     string `gorm:"type:text" json:"title"`
	Summary   string `gorm:"type:text" json:"summary"`
	Content   string `gorm:"type:text;not null" json:"content"`
	// Origin 快照来源: manual / ai_outline / ai_draft / ai_refine / pre_restore
	Origin    string    `gorm:"size:16;not null" json:"origin"`
	Note      string    `gorm:"type:text" json:"note"`
	CreatedBy uint64    `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (ArticleVersion) TableName() string { return "article_versions" }
