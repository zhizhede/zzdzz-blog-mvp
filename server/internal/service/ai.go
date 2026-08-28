package service

import (
	"errors"
	"unicode/utf8"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
)

const (
	defaultConvTitle     = "未命名会话"
	autoTitleMaxRunes    = 20
	autoTitleTriggerMsgs = 2 // 首条 user + 首条 assistant 完成后尝试起标题
)

type AIService struct {
	db *gorm.DB
}

func NewAIService(db *gorm.DB) *AIService {
	return &AIService{db: db}
}

func (s *AIService) ListConversations(userID uint64) ([]model.AIConversation, error) {
	var items []model.AIConversation
	if err := s.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *AIService) GetConversation(userID, convID uint64) (*model.AIConversation, error) {
	var c model.AIConversation
	if err := s.db.Where("id = ? AND user_id = ?", convID, userID).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *AIService) CreateConversation(userID uint64) (*model.AIConversation, error) {
	c := &model.AIConversation{
		UserID: userID,
		Title:  defaultConvTitle,
	}
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *AIService) RenameConversation(userID, convID uint64, title string) error {
	title = trimTitle(title)
	if title == "" {
		return nil
	}
	res := s.db.Model(&model.AIConversation{}).
		Where("id = ? AND user_id = ?", convID, userID).
		Update("title", title)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	return nil
}

func (s *AIService) DeleteConversation(userID, convID uint64) error {
	res := s.db.Where("id = ? AND user_id = ?", convID, userID).
		Delete(&model.AIConversation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	return nil
}

// GetRecentMessages 取最近 limit 条消息,按时间升序返回,用于构造 LLM 上下文。
func (s *AIService) GetRecentMessages(userID, convID uint64, limit int) ([]model.AIMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	if _, err := s.GetConversation(userID, convID); err != nil {
		return nil, err
	}
	var msgs []model.AIMessage
	if err := s.db.Where("conversation_id = ?", convID).
		Order("created_at DESC").
		Limit(limit).
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	// 反转成升序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// AppendUserMessage 用户消息立即入库,并把会话 updated_at 顶到最新。
func (s *AIService) AppendUserMessage(convID uint64, content string) (*model.AIMessage, error) {
	m := &model.AIMessage{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	s.db.Model(&model.AIConversation{}).
		Where("id = ?", convID).
		UpdateColumn("updated_at", gorm.Expr("NOW()"))
	return m, nil
}

// CreateAssistantPlaceholder 流式开始时插一条 content='' 的占位行,返回 msg id。
func (s *AIService) CreateAssistantPlaceholder(convID uint64) (uint64, error) {
	m := &model.AIMessage{
		ConversationID: convID,
		Role:           "assistant",
		Content:        "",
	}
	if err := s.db.Create(m).Error; err != nil {
		return 0, err
	}
	return m.ID, nil
}

// AppendDelta SSE 每个 chunk 来时调用,把 delta 拼接到对应 assistant 消息。
// 使用 PostgreSQL 的 || 文本拼接,数据库侧原子。
func (s *AIService) AppendDelta(msgID uint64, delta string) error {
	if delta == "" {
		return nil
	}
	return s.db.Exec(
		`UPDATE ai_messages SET content = content || ? WHERE id = ?`,
		delta, msgID,
	).Error
}

// FinalizeAssistant 流式结束后调用,内容已经逐 delta 入库,这里只做:
//  1. 必要时(根据首条 user 消息)自动起标题
//  2. 顶一下会话 updated_at
func (s *AIService) FinalizeAssistant(userID, convID, msgID uint64) error {
	var msg model.AIMessage
	if err := s.db.First(&msg, msgID).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&model.AIMessage{}).
		Where("conversation_id = ?", convID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == int64(autoTitleTriggerMsgs) {
		// 第一次完成首轮对话,尝试起标题
		var firstUser model.AIMessage
		if err := s.db.Where("conversation_id = ? AND role = ?", convID, "user").
			Order("created_at ASC").
			First(&firstUser).Error; err == nil {
			s.maybeAutoTitle(userID, convID, firstUser.Content)
		}
	}
	// 不管是否起标题,都把 updated_at 推到最新,列表排序才正确
	return s.db.Model(&model.AIConversation{}).
		Where("id = ?", convID).
		UpdateColumn("updated_at", gorm.Expr("NOW()")).Error
}

func (s *AIService) maybeAutoTitle(userID, convID uint64, firstUserContent string) {
	var c model.AIConversation
	if err := s.db.Where("id = ? AND user_id = ?", convID, userID).
		First(&c).Error; err != nil {
		return
	}
	if c.Title != defaultConvTitle {
		return
	}
	title := trimTitle(firstUserContent)
	if title == "" {
		return
	}
	s.db.Model(&model.AIConversation{}).
		Where("id = ? AND user_id = ?", convID, userID).
		Update("title", title)
}

func trimTitle(s string) string {
	s = truncateRunes(s, autoTitleMaxRunes)
	return s
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}