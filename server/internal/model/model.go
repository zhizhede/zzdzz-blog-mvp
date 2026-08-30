package model

import "time"

type Base struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ConversationID uint64    `gorm:"index;not null" json:"conversation_id"`
	Role           string    `gorm:"size:16;not null" json:"role"`
	Content        string    `gorm:"type:text;not null;default:''" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AIMessage) TableName() string { return "ai_messages" }