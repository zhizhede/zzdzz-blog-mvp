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
		response.Unauthorized(c, "invalid username or password")
		return
	}

	response.OK(c, gin.H{
		"token":      token,
		"user":       gin.H{"id": user.ID, "username": user.Username},
		"expires_in": int(h.cfg.ExpireDuration().Seconds()),
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("user_id")
	uname, _ := c.Get("username")
	response.OK(c, gin.H{"id": uid, "username": uname})
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
		c.Next()
	}
}