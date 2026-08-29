package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/service"
	jwtutil "zzdzz-blog/server/pkg/jwt"
	"zzdzz-blog/server/pkg/response"
)

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.JWTConfig
}

func NewAuthHandler(svc *service.AuthService, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "username and password required")
		return
	}

	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		// 不区分"密码错误"和"账号被禁用", 避免泄漏账号存在性
		response.Unauthorized(c, "invalid username or password")
		return
	}

	response.OK(c, gin.H{
		"token":      token,
		"user":       gin.H{"id": user.ID, "username": user.Username, "is_admin": user.IsAdmin},
		"expires_in": int(h.cfg.ExpireDuration().Seconds()),
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("user_id")
	uname, _ := c.Get("username")
	isAdmin, _ := c.Get("is_admin")
	response.OK(c, gin.H{"id": uid, "username": uname, "is_admin": isAdmin})
}

// userIDOf 从 gin.Context 取 user_id,支持 uint64 / float64(JSON 解码) / int 三种形态。
func userIDOf(c *gin.Context) uint64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case uint64:
		return x
	case int:
		return uint64(x)
	case int64:
		return uint64(x)
	case float64:
		return uint64(x)
	}
	return 0
}

func RequireAuth(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Unauthorized(c, "missing bearer token")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims, err := jwtutil.Parse(cfg.Secret, tokenStr)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Next()
	}
}

// RequireAdmin 在 RequireAuth 基础上进一步校验 is_admin=true.
// 必须挂在 RequireAuth 之后(或调用 RequireAuth 自身, 因为它已经注入 is_admin).
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("is_admin")
		isAdmin, _ := v.(bool)
		if !ok || !isAdmin {
			response.Fail(c, 403, 4003, "admin only")
			c.Abort()
			return
		}
		c.Next()
	}
}

// OptionalAuth 复用 RequireAuth 的解析逻辑, 但 token 缺失/无效时放行:
//   - 有 token 且有效: 注入 user_id / username / is_admin, handler 可识别为已登录用户
//   - 无 token 或 token 无效: 跳过, 继续走后续 handler(可能用于公开接口的"能识别就识别")
func OptionalAuth(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.Next()
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims, err := jwtutil.Parse(cfg.Secret, tokenStr)
		if err != nil {
			c.Next()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Next()
	}
}