package service

import (
	"time"

	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

type VisitLogService struct {
	db *gorm.DB
}

func NewVisitLogService(db *gorm.DB) *VisitLogService {
	return &VisitLogService{db: db}
}

type VisitLogListResult struct {
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Items []model.VisitLog `json:"items"`
}

// HasVisitSince 该 IP 自 since 起是否已有记录, 当天去重的依据
func (s *VisitLogService) HasVisitSince(ip string, since time.Time) (bool, error) {
	var count int64
	if err := s.db.Model(&model.VisitLog{}).
		Where("ip = ? AND created_at >= ?", ip, since).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Record 插入一条访问记录, 中间件异步调用
func (s *VisitLogService) Record(v *model.VisitLog) error {
	return s.db.Create(v).Error
}

// List 分页查询访问记录, ip 非空时精确过滤, 按 created_at 倒序
func (s *VisitLogService) List(page, pageSize int, ip string) (*VisitLogListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	tx := s.db.Model(&model.VisitLog{})
	if ip != "" {
		tx = tx.Where("ip = ?", ip)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []model.VisitLog
	if err := tx.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &VisitLogListResult{
		Total: total,
		Page:  page,
		Size:  pageSize,
		Items: items,
	}, nil
}
