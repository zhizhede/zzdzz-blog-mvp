package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/pkg/response"
)

// rateLimiter 按 key(客户端 IP) 的固定窗口计数器, 进程内内存实现.
// 单实例部署(宝塔单机), 无需分布式.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	limit   int
	window  time.Duration
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

const rateSweepThreshold = 4096

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		entries: make(map[string]rateEntry),
		limit:   limit,
		window:  window,
	}
}

// allow 返回本次是否放行. 窗口过期自动重置; 条目超过阈值时顺带清理过期项,
// 防止恶意刷接口把内存撑大.
func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key]
	if !ok || !now.Before(e.resetAt) {
		if len(l.entries) > rateSweepThreshold {
			for k, v := range l.entries {
				if !now.Before(v.resetAt) {
					delete(l.entries, k)
				}
			}
		}
		l.entries[key] = rateEntry{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if e.count >= l.limit {
		return false
	}
	e.count++
	l.entries[key] = e
	return true
}

// RateLimit 按客户端 IP 限流中间件: window 内最多 limit 次, 超出返回 429.
// 挂在路由上时每次调用生成独立计数器(登录/注册各自计数).
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	l := newRateLimiter(limit, window)
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP()) {
			response.Fail(c, http.StatusTooManyRequests, 4029, "请求过于频繁, 请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
