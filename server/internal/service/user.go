package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"zzdzz-blog/server/internal/model"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameConflict   = errors.New("username already exists")
	ErrCannotDisableSelf  = errors.New("cannot disable your own account")
	ErrLastActiveUser     = errors.New("cannot disable the last active user")
	ErrInvalidOldPassword = errors.New("invalid old password")
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// List 返回所有用户, 不含 password_hash
func (s *UserService) List() ([]model.User, error) {
	var users []model.User
	if err := s.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create 新建用户, 密码 bcrypt 哈希入库
func (s *UserService) Create(username, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		IsActive:     true,
	}
	if err := s.db.Create(u).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameConflict
		}
		return nil, err
	}
	return u, nil
}

// ChangePassword 修改密码: 需要旧密码(任何登录者改自己的密码都要校验旧密码)
func (s *UserService) ChangePassword(actorID, targetID uint64, oldPassword, newPassword string) error {
	var u model.User
	if err := s.db.First(&u, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	// 只能改自己的密码; 不开放管理员改任意用户密码的接口(避免误改 admin)
	if actorID != targetID {
		return ErrUserNotFound
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

// SetActive 启用/禁用用户
func (s *UserService) SetActive(actorID, targetID uint64, isActive bool) error {
	var u model.User
	if err := s.db.First(&u, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 不允许禁用自己
	if actorID == targetID && !isActive {
		return ErrCannotDisableSelf
	}

	// 不允许禁用最后一个启用账户(防止锁死系统)
	if !isActive && u.IsActive {
		var activeCount int64
		if err := s.db.Model(&model.User{}).Where("is_active = ?", true).Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount <= 1 {
			return ErrLastActiveUser
		}
	}

	if u.IsActive == isActive {
		return nil
	}
	return s.db.Model(&u).Update("is_active", isActive).Error
}