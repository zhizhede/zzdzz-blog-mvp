package handler

import (
	"errors"
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
		response.BadRequest(c, "请输入用户名和密码")
		return
	}

	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		// 不区分"密码错误"和"账号被禁用", 避免泄漏账号存在性
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	response.OK(c, gin.H{
		"token":      token,
		"user":       gin.H{"id": user.ID, "uuid": user.UUID, "username": user.Username, "is_admin": user.IsAdmin},
		"expires_in": int(h.cfg.ExpireDuration().Seconds()),
	})
}

type registerReq struct {
	Username     string `json:"username" binding:"required,min=3,max=64"`
	Password     string `json:"password" binding:"required,min=6,max=64"`
	PasswordHint string `json:"password_hint" binding:"max=255"`
}

// Register 开放注册: 只需用户名 + 密码(用户名全系统唯一, 0014); 成功即返回 token(注册即登录).
// password_hint 选填(0015), 忘记密码时可通过 /auth/password-hint 查看.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "用户名需 3-64 字符, 密码需 6-64 字符")
		return
	}

	token, user, err := h.svc.Register(req.Username, req.Password, req.PasswordHint)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameConflict):
			response.Fail(c, 409, 4009, "用户名已存在")
		case errors.Is(err, service.ErrUsernameReserved):
			response.Fail(c, 409, 4010, "该用户名为保留名, 无法使用")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}

	response.OK(c, gin.H{
		"token":      token,
		"user":       gin.H{"id": user.ID, "uuid": user.UUID, "username": user.Username, "is_admin": user.IsAdmin},
		"expires_in": int(h.cfg.ExpireDuration().Seconds()),
	})
}

// Me 页面刷新时拉最新用户状态. uuid 不进 JWT(老 token 无此字段), 每次按 id 现查.
func (h *AuthHandler) Me(c *gin.Context) {
	uid := userIDOf(c)
	uname, _ := c.Get("username")
	isAdmin, _ := c.Get("is_admin")
	uuid := ""
	hint := ""
	if u, err := h.svc.GetByID(uid); err == nil {
		uuid = u.UUID
		hint = u.PasswordHint
	}
	response.OK(c, gin.H{"id": uid, "uuid": uuid, "username": uname, "is_admin": isAdmin, "password_hint": hint})
}

type changeOwnPasswordHintReq struct {
	Password     string `json:"password" binding:"required"`
	PasswordHint string `json:"password_hint" binding:"max=255"`
}

// ChangeOwnPasswordHint 登录用户改自己的密码提示(0015). 必须携带当前密码,
// 验证通过才允许修改; 提示可为空串(表示清除提示).
func (h *AuthHandler) ChangeOwnPasswordHint(c *gin.Context) {
	uid := userIDOf(c)
	if uid == 0 {
		response.Unauthorized(c, "缺少用户身份")
		return
	}
	var req changeOwnPasswordHintReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入当前密码, 提示最长 255 字")
		return
	}
	if err := h.svc.ChangePasswordHint(uid, req.Password, req.PasswordHint); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, 404, 4004, "用户不存在")
		case errors.Is(err, service.ErrInvalidOldPassword):
			response.Fail(c, 400, 4001, "密码不正确")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
}

type passwordHintReq struct {
	Username string `json:"username" binding:"required"`
}

// PasswordHint 忘记密码时按用户名查密码提示(0015). 公开接口(未登录态),
// 用户不存在时同样返回空串, 不区分"无此用户"和"未设置提示".
func (h *AuthHandler) PasswordHint(c *gin.Context) {
	var req passwordHintReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入用户名")
		return
	}
	response.OK(c, gin.H{"username": req.Username, "password_hint": h.svc.PasswordHint(req.Username)})
}

type changeOwnPasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// ChangeOwnPassword 登录用户自助改密. 只能改自己的密码; handler 强制 actorID == targetID.
func (h *AuthHandler) ChangeOwnPassword(c *gin.Context) {
	uid := userIDOf(c)
	if uid == 0 {
		response.Unauthorized(c, "缺少用户身份")
		return
	}
	var req changeOwnPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入旧密码, 新密码至少 6 位")
		return
	}
	if err := h.svc.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, 404, 4004, "用户不存在")
		case errors.Is(err, service.ErrInvalidOldPassword):
			response.Fail(c, 400, 4001, "旧密码不正确")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
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
			response.Unauthorized(c, "缺少登录凭证")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims, err := jwtutil.Parse(cfg.Secret, tokenStr)
		if err != nil {
			response.Unauthorized(c, "登录已失效, 请重新登录")
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