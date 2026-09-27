package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

// 快捷用语板(0019): 用户自预设短语, 笔记/文章编辑器点击即插入正文光标处.
// 纯 CRUD, 一切查询都强制带 user_id 条件; id 属主不符按"不存在"处理,
// 不向外泄露该 id 是否属于其他用户.

const (
	// MaxQuickPhraseLen 单条短语长度上限(按 rune 计)
	MaxQuickPhraseLen = 200
	// MaxQuickPhrasesPerUser 每用户短语条数上限, 防刷
	MaxQuickPhrasesPerUser = 100
)

var (
	// ErrQuickPhraseNotFound 短语不存在或不属于当前用户
	ErrQuickPhraseNotFound = errors.New("快捷用语不存在")
	ErrQuickPhraseEmpty    = errors.New("快捷用语内容不能为空")
	ErrQuickPhraseTooLong  = errors.New("快捷用语过长(最多 200 字)")
	ErrQuickPhraseTooMany  = errors.New("快捷用语已达上限(100 条), 请先删除部分再添加")
)

type QuickPhraseService struct {
	db *gorm.DB
}

func NewQuickPhraseService(db *gorm.DB) *QuickPhraseService {
	return &QuickPhraseService{db: db}
}

// List 当前用户的短语, 按排序值与创建顺序稳定输出
func (s *QuickPhraseService) List(userID uint64) ([]model.QuickPhrase, error) {
	var ps []model.QuickPhrase
	err := s.db.Where("user_id = ?", userID).
		Order("sort_order ASC, id ASC").Find(&ps).Error
	return ps, err
}

func (s *QuickPhraseService) Create(userID uint64, content string) (*model.QuickPhrase, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrQuickPhraseEmpty
	}
	if utf8.RuneCountInString(content) > MaxQuickPhraseLen {
		return nil, ErrQuickPhraseTooLong
	}
	var n int64
	if err := s.db.Model(&model.QuickPhrase{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return nil, err
	}
	if n >= MaxQuickPhrasesPerUser {
		return nil, ErrQuickPhraseTooMany
	}
	p := model.QuickPhrase{UserID: userID, Content: content, SortOrder: int(n)}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Update 改内容; 属主不符按不存在处理
func (s *QuickPhraseService) Update(userID, id uint64, content string) (*model.QuickPhrase, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrQuickPhraseEmpty
	}
	if utf8.RuneCountInString(content) > MaxQuickPhraseLen {
		return nil, ErrQuickPhraseTooLong
	}
	var p model.QuickPhrase
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrQuickPhraseNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Content = content
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete 删除; 属主不符按不存在处理
func (s *QuickPhraseService) Delete(userID, id uint64) error {
	res := s.db.Where("user_id = ?", userID).Delete(&model.QuickPhrase{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrQuickPhraseNotFound
	}
	return nil
}
