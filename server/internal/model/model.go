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
}

func (Article) TableName() string { return "articles" }

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