package service

import (
	"fmt"
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

// AttributeOrRecord 当天归属逻辑:
//   - 该 IP 当天已有记录且为匿名(user_id IS NULL): 带 uid 时回填归属(访客登录后
//     把当天记录认领回来), 已归属其他用户的记录绝不覆盖;
//   - 当天无记录: 插入新记录(带或不带 uid).
//
// 并发竞态下极小概率重复插入, 无害(0016 迁移注释有述).
func (s *VisitLogService) AttributeOrRecord(ip string, uid *uint64, path, ua string, dayStart time.Time) error {
	var anon int64
	if err := s.db.Model(&model.VisitLog{}).
		Where("ip = ? AND created_at >= ? AND user_id IS NULL", ip, dayStart).
		Count(&anon).Error; err != nil {
		return err
	}
	if anon > 0 {
		if uid == nil {
			return nil
		}
		return s.db.Model(&model.VisitLog{}).
			Where("ip = ? AND created_at >= ? AND user_id IS NULL", ip, dayStart).
			Update("user_id", *uid).Error
	}
	return s.db.Create(&model.VisitLog{IP: ip, UserID: uid, Path: path, UserAgent: ua}).Error
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

// StatsForAI 生成注入 AI 对话的实时访问摘要(仅 admin 请求时由 handler 注入).
// 数据全部来自数据库实查, AI 据此回答"今天多少人访问/最近谁来了"类问题.
func (s *VisitLogService) StatsForAI() (string, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	var todayCnt, todayIPs, yCnt, yIPs, totalCnt, totalIPs int64
	if err := s.db.Model(&model.VisitLog{}).Where("created_at >= ?", todayStart).Count(&todayCnt).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&model.VisitLog{}).Where("created_at >= ?", todayStart).Distinct("ip").Count(&todayIPs).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&model.VisitLog{}).
		Where("created_at >= ? AND created_at < ?", yesterdayStart, todayStart).
		Count(&yCnt).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&model.VisitLog{}).
		Where("created_at >= ? AND created_at < ?", yesterdayStart, todayStart).
		Distinct("ip").Count(&yIPs).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&model.VisitLog{}).Count(&totalCnt).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&model.VisitLog{}).Distinct("ip").Count(&totalIPs).Error; err != nil {
		return "", err
	}

	var recent []model.VisitLog
	if err := s.db.Order("created_at DESC").Limit(10).Find(&recent).Error; err != nil {
		return "", err
	}

	var topPaths []struct {
		Path string `gorm:"column:path"`
		N    int64  `gorm:"column:n"`
	}
	if err := s.db.Model(&model.VisitLog{}).
		Select("path, COUNT(*) AS n").Group("path").Order("n DESC").Limit(5).
		Scan(&topPaths).Error; err != nil {
		return "", err
	}

	var topUAs []struct {
		UA string `gorm:"column:ua"`
		N  int64  `gorm:"column:n"`
	}
	if err := s.db.Model(&model.VisitLog{}).
		Select("user_agent AS ua, COUNT(*) AS n").Group("user_agent").Order("n DESC").Limit(3).
		Scan(&topUAs).Error; err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("【网站实时访问统计】以下是服务器刚从数据库查到的真实数据, 回答访问相关问题必须以此为准, 不要编造:\n")
	b.WriteString(fmt.Sprintf("- 今日(%s): %d 条记录 / %d 个独立访客 IP\n", now.Format("01-02"), todayCnt, todayIPs))
	b.WriteString(fmt.Sprintf("- 昨日: %d 条记录 / %d 个独立访客 IP\n", yCnt, yIPs))
	b.WriteString(fmt.Sprintf("- 累计: %d 条记录 / %d 个独立访客 IP\n", totalCnt, totalIPs))
	b.WriteString("- 最近 10 条访问(时间 | IP | 用户 | 路径):\n")
	for _, v := range recent {
		user := "匿名"
		if v.UserID != nil {
			user = fmt.Sprintf("#%d", *v.UserID)
		}
		b.WriteString(fmt.Sprintf("  %s | %s | %s | %s\n",
			v.CreatedAt.Format("01-02 15:04"), v.IP, user, v.Path))
	}
	pathParts := make([]string, 0, len(topPaths))
	for _, p := range topPaths {
		pathParts = append(pathParts, fmt.Sprintf("%s(%d次)", p.Path, p.N))
	}
	b.WriteString("- 访问最多路径 Top5: "+strings.Join(pathParts, ", ")+"\n")
	uaParts := make([]string, 0, len(topUAs))
	for _, u := range topUAs {
		uaParts = append(uaParts, fmt.Sprintf("%s(%d次)", truncateUA(u.UA, 40), u.N))
	}
	b.WriteString("- 常见 UA Top3: "+strings.Join(uaParts, ", ")+"\n")
	b.WriteString("说明: 每位访客(IP)每天只记一条; 路径为 .php/.env/wp-login 等非常规页面多为互联网扫描器自动探测, 属正常背景噪音, 不是真实访客.\n")
	return b.String(), nil
}

// truncateUA 截断过长的 UA, 避免摘要膨胀
func truncateUA(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
