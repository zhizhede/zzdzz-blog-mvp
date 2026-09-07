package service

import (
	"errors"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
	jwtutil "zzdzz-blog/server/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrUsernameReserved   = errors.New("username is reserved")
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

// Login username 全系统唯一(0014): 按用户名取唯一未删除行比对密码.
// "密码错误"和"账号被禁用"对外不区分(handler 统一 401), 这里区分是为了禁用语义正确.
func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var u model.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
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

// Register 开放注册: 用户名全系统唯一(0014 恢复), 重名返回 ErrUsernameConflict.
// 保留名(ZZDZZ_ADMIN_USERNAMES, 默认 "admin")拒绝注册, 防止有人抢注 admin 身份.
func (s *AuthService) Register(username, password string) (string, *model.User, error) {
	if _, reserved := adminUsernames()[username]; reserved {
		return "", nil, ErrUsernameReserved
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}
	u := &model.User{
		UUID:         uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		IsActive:     true,
	}
	if err := s.db.Create(u).Error; err != nil {
		// 并发注册同名靠 uq_users_username_alive 唯一索引兜底, 这里统一转 409
		if isUniqueViolation(err) {
			return "", nil, ErrUsernameConflict
		}
		return "", nil, err
	}
	token, err := jwtutil.Generate(s.cfg.Secret, s.cfg.ExpireDuration(), u.ID, u.Username, false)
	if err != nil {
		return "", nil, err
	}
	u.IsAdmin = false
	return token, u, nil
}

// GetByID 按 id 查用户, 供 /auth/me 回填 uuid 等场景
func (s *AuthService) GetByID(id uint64) (*model.User, error) {
	var u model.User
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ChangePassword 登录用户自助改密. 强制 actorID == targetID, 由 handler 自行保证.
func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	var u model.User
	if err := s.db.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidOldPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Model(&u).Update("password_hash", string(hash)).Error
}