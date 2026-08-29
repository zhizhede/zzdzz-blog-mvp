package service

import (
	"errors"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
	jwtutil "zzdzz-blog/server/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user is disabled")
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.JWTConfig
}

func NewAuthService(db *gorm.DB, cfg *config.JWTConfig) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

// adminUsernames 返回超级管理员用户名集合. 本期最小实现: 环境变量 ZZDZZ_ADMIN_USERNAMES
// (逗号分隔, 默认 "admin"). 后续要加 is_admin 列时再切换.
func adminUsernames() map[string]struct{} {
	raw := os.Getenv("ZZDZZ_ADMIN_USERNAMES")
	if raw == "" {
		raw = "admin"
	}
	set := map[string]struct{}{}
	cur := ""
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			if cur != "" {
				set[cur] = struct{}{}
				cur = ""
			}
			continue
		}
		cur += string(raw[i])
	}
	return set
}

func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var u model.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		return "", nil, ErrUserDisabled
	}

	_, isAdmin := adminUsernames()[u.Username]
	token, err := jwtutil.Generate(s.cfg.Secret, s.cfg.ExpireDuration(), u.ID, u.Username, isAdmin)
	if err != nil {
		return "", nil, err
	}
	// 回填给调用方, 让 /auth/login 响应里 is_admin 字段正确
	u.IsAdmin = isAdmin
	return token, &u, nil
}