package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", RateLimit(3, time.Hour), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip + ":12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 同 IP 3 次放行, 第 4 次 429
	for i := 1; i <= 3; i++ {
		if code := do("1.2.3.4"); code != http.StatusOK {
			t.Fatalf("第 %d 次请求 = %d, want 200", i, code)
		}
	}
	if code := do("1.2.3.4"); code != http.StatusTooManyRequests {
		t.Errorf("超限请求 = %d, want 429", code)
	}
	// 不同 IP 不受影响
	if code := do("5.6.7.8"); code != http.StatusOK {
		t.Errorf("其他 IP = %d, want 200", code)
	}
}

func TestRateLimiterWindowReset(t *testing.T) {
	l := newRateLimiter(1, 10*time.Millisecond)
	if !l.allow("k") {
		t.Fatal("首次应放行")
	}
	if l.allow("k") {
		t.Fatal("超限应拒绝")
	}
	time.Sleep(15 * time.Millisecond)
	if !l.allow("k") {
		t.Fatal("窗口过期后应重新放行")
	}
}
