package service

import (
	"strconv"
	"strings"
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

// VisitLogRow 列表行: LEFT JOIN users 附带用户名(用户已删/不存在时为 NULL)
type VisitLogRow struct {
	model.VisitLog
	Username *string `json:"username"`
}

type VisitLogListResult struct {
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Items []VisitLogRow  `json:"items"`
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

// List 分页查询访问记录: ip 精确匹配, path/ua 为包含匹配(ILIKE), 按 created_at 倒序.
// userFilter 语义: "匿名" → 匿名访客(user_id IS NULL) ∪ 用户名叫"匿名"的注册用户;
// "#数字" → 按用户 ID 精确; 其余文本 → 按用户名模糊匹配.
func (s *VisitLogService) List(page, pageSize int, ip, userFilter, pathKeyword, uaKeyword string) (*VisitLogListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	tx := s.db.Model(&model.VisitLog{})
	joined := false
	if ip != "" {
		tx = tx.Where("ip = ?", ip)
	}
	if userFilter != "" {
		tx = tx.Joins("LEFT JOIN users ON users.id = visit_logs.user_id")
		joined = true
		switch {
		case userFilter == "匿名":
			// 两类都要, 展示层用徽标形态区分(虚线"匿名" vs 强调色"#ID 用户名")
			tx = tx.Where("visit_logs.user_id IS NULL OR users.username = ?", userFilter)
		case strings.HasPrefix(userFilter, "#"):
			id, err := strconv.ParseUint(strings.TrimPrefix(userFilter, "#"), 10, 64)
			if err != nil {
				// "#xxx" 不是数字 → 无匹配
				tx = tx.Where("1 = 0")
			} else {
				tx = tx.Where("visit_logs.user_id = ?", id)
			}
		default:
			tx = tx.Where("users.username ILIKE ?", "%"+userFilter+"%")
		}
	}
	if pathKeyword != "" {
		tx = tx.Where("path ILIKE ?", "%"+pathKeyword+"%")
	}
	if uaKeyword != "" {
		tx = tx.Where("user_agent ILIKE ?", "%"+uaKeyword+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	selectCols := "visit_logs.*"
	if joined {
		selectCols += ", users.username AS username"
	}
	var items []VisitLogRow
	if err := tx.Select(selectCols).
		Order("visit_logs.created_at DESC").
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
