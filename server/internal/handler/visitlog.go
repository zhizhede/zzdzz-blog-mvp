package handler

import (
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
	"zzdzz-blog/server/internal/service"
	jwtutil "zzdzz-blog/server/pkg/jwt"
	"zzdzz-blog/server/pkg/response"
)

type VisitLogHandler struct {
	svc *service.VisitLogService
}

func NewVisitLogHandler(svc *service.VisitLogService) *VisitLogHandler {
	return &VisitLogHandler{svc: svc}
}

// RecordVisit 全局访问记录中间件(0016): 每位访客(IP)每天至多一条.
// 带 Bearer token 且有效时附带 user_id, 否则匿名(NULL).
// 必须在路由注册之前 r.Use, 否则静态页/NoRoute 请求不经过本中间件.
func RecordVisit(db *gorm.DB, jwtSecret string) gin.HandlerFunc {
	svc := service.NewVisitLogService(db)
	return func(c *gin.Context) {
		// 只记读取类请求(页面/API 拉取), 写操作算"动作"不算"到访";
		// 预检/探活/静态资源不记, 保证首条记录落在页面或 API 入口上
		method := c.Request.Method
		if (method != http.MethodGet && method != http.MethodHead) ||
			isNoisePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			c.Next()
			return
		}
		// goroutine 里不能再用 c, 请求生命周期内先把字段取干净
		uid := visitUserID(jwtSecret, c.GetHeader("Authorization"))
		p := truncateRunes(c.Request.URL.Path, 512)
		ua := truncateRunes(c.Request.UserAgent(), 512)

		go func() {
			now := time.Now()
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			seen, err := svc.HasVisitSince(ip, midnight)
			if err != nil {
				log.Printf("[visit] 当天去重查询失败: %v", err)
				return
			}
			if seen {
				return
			}
			v := &model.VisitLog{IP: ip, UserID: uid, Path: p, UserAgent: ua}
			if err := svc.Record(v); err != nil {
				log.Printf("[visit] 记录访问失败: %v", err)
			}
		}()

		c.Next()
	}
}

// visitUserID 从 Bearer token 解析访客用户 ID; 无 token/无效/过期一律按匿名(nil),
// 与 OptionalAuth 的宽容语义一致, 记录访问绝不因 token 问题失败.
func visitUserID(secret, header string) *uint64 {
	if !strings.HasPrefix(header, "Bearer ") {
		return nil
	}
	claims, err := jwtutil.Parse(secret, strings.TrimPrefix(header, "Bearer "))
	if err != nil || claims.UserID == 0 {
		return nil
	}
	return &claims.UserID
}

// isNoisePath 排除不值得记录的路径: 探活接口与静态资源(favicon 系列也走静态后缀过滤)
func isNoisePath(p string) bool {
	if p == "/api/v1/ping" {
		return true
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".js", ".css", ".map", ".png", ".jpg", ".jpeg", ".gif", ".ico",
		".svg", ".woff", ".woff2", ".ttf", ".txt", ".webp", ".webmanifest":
		return true
	}
	return false
}

// truncateRunes 按字符截断到 n 个, PG 的 VARCHAR(n) 按字符计长;
// 按字节切会把多字节字符切成非法 UTF-8 被 PG 拒收
func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// List admin 分页查询访问记录: GET /api/v1/visit-logs?page=&page_size=&ip=
func (h *VisitLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.svc.List(page, pageSize, c.Query("ip"))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, result)
}
