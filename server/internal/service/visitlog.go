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

// Record 无差别记录(0018 起不再按 IP+天去重): 每个通过噪音过滤的请求
// 各记一条, 归属随请求自身的登录态(页面加载为匿名, 登录 API 请求带 user_id).
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

	// 用户 ID→用户名对照(用户少, 全量带出), AI 才能把 "#8" 和 "zzdzz" 对上号
	type userRow struct {
		ID       uint64
		Username string
	}
	var users []userRow
	if err := s.db.Model(&model.User{}).Select("id, username").Order("id").Scan(&users).Error; err != nil {
		return "", err
	}
	nameOf := make(map[uint64]string, len(users))
	mapParts := make([]string, 0, len(users))
	for _, u := range users {
		nameOf[u.ID] = u.Username
		mapParts = append(mapParts, fmt.Sprintf("#%d=%s", u.ID, u.Username))
	}

	// 按用户统计(今日/累计), 匿名单独一行
	type userCnt struct {
		UserID  *uint64 `gorm:"column:user_id"`
		N       int64   `gorm:"column:n"`
	}
	countByUser := func(todayOnly bool) (map[uint64]int64, int64, error) {
		q := s.db.Model(&model.VisitLog{}).Select("user_id, COUNT(*) AS n").Group("user_id")
		if todayOnly {
			q = q.Where("created_at >= ?", todayStart)
		}
		var rows []userCnt
		if err := q.Scan(&rows).Error; err != nil {
			return nil, 0, err
		}
		m := make(map[uint64]int64, len(rows))
		var anon int64
		for _, r := range rows {
			if r.UserID != nil {
				m[*r.UserID] = r.N
			} else {
				anon = r.N
			}
		}
		return m, anon, nil
	}
	todayByUser, todayAnon, err := countByUser(true)
	if err != nil {
		return "", err
	}
	totalByUser, totalAnon, err := countByUser(false)
	if err != nil {
		return "", err
	}

	// 每个注册用户最近一条到访(DISTINCT ON 取每人最新一行), 让"某用户来过吗"
	// 这类问题有现成答案, 不依赖模型自己从最近 10 条里推断
	type lastVisit struct {
		UserID    uint64    `gorm:"column:user_id"`
		IP        string    `gorm:"column:ip"`
		Path      string    `gorm:"column:path"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var lasts []lastVisit
	if err := s.db.Raw("SELECT DISTINCT ON (user_id) user_id, ip, path, created_at "+
		"FROM visit_logs WHERE user_id IS NOT NULL ORDER BY user_id, created_at DESC").
		Scan(&lasts).Error; err != nil {
		return "", err
	}
	lastOf := make(map[uint64]lastVisit, len(lasts))
	for _, l := range lasts {
		lastOf[l.UserID] = l
	}

	var b strings.Builder
	b.WriteString("【网站实时访问统计】以下是服务器刚从数据库查到的真实数据, 回答访问相关问题必须以此为准, 不要编造:\n")
	b.WriteString(fmt.Sprintf("- 今日(%s): %d 条记录 / %d 个独立访客 IP\n", now.Format("01-02"), todayCnt, todayIPs))
	b.WriteString(fmt.Sprintf("- 昨日: %d 条记录 / %d 个独立访客 IP\n", yCnt, yIPs))
	b.WriteString(fmt.Sprintf("- 累计: %d 条记录 / %d 个独立访客 IP\n", totalCnt, totalIPs))
	b.WriteString("- 用户对照: "+strings.Join(mapParts, ", ")+"\n")
	statParts := make([]string, 0, len(users)+1)
	for _, u := range users {
		statParts = append(statParts, fmt.Sprintf("%s(#%d) %d/%d", u.Username, u.ID, todayByUser[u.ID], totalByUser[u.ID]))
	}
	statParts = append(statParts, fmt.Sprintf("匿名 %d/%d", todayAnon, totalAnon))
	b.WriteString("- 各用户记录数(今日/累计): "+strings.Join(statParts, "; ")+"\n")
	lastParts := make([]string, 0, len(users))
	for _, u := range users {
		if l, ok := lastOf[u.ID]; ok {
			lastParts = append(lastParts, fmt.Sprintf("%s(#%d) %s @%s %s",
				u.Username, u.ID, l.CreatedAt.Format("01-02 15:04"), l.IP, l.Path))
		} else {
			lastParts = append(lastParts, fmt.Sprintf("%s(#%d) 从未到访", u.Username, u.ID))
		}
	}
	b.WriteString("- 各注册用户最近一条到访: "+strings.Join(lastParts, "; ")+"\n")
	b.WriteString("- 最近 10 条访问(时间 | IP | 用户 | 路径):\n")
	for _, v := range recent {
		user := "匿名"
		if v.UserID != nil {
			user = fmt.Sprintf("#%d", *v.UserID)
			if name, ok := nameOf[*v.UserID]; ok {
				user += " " + name
			}
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
	b.WriteString("说明: 无差别记录每个请求, 同一次浏览会产生多条(页面加载记匿名, 登录后的 API 请求带用户), 条数是请求数不是人数, 真实访客数看独立 IP 数; 路径为 .php/.env/wp-login 等非常规页面多为互联网扫描器自动探测, 属正常背景噪音, 不是真实访客.\n")
	b.WriteString("重要: 回答某用户的访问问题时, 先用上方「用户对照」把用户名换成 #ID, 再引用「各用户记录数」和「各注册用户最近一条到访」作答; 最近 10 条里没出现该用户不代表该用户没有到访记录.\n")
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
