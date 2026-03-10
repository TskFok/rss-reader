package services

import (
	"errors"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
)

// AdminService 管理员服务
type AdminService struct {
	db *gorm.DB
}

// NewAdminService 创建管理员服务
func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
}

// ListUsers 获取用户列表（不含密码）
func (s *AdminService) ListUsers() ([]models.User, error) {
	var users []models.User
	err := s.db.Order("created_at DESC").Find(&users).Error
	return users, err
}

// UnlockUser 解锁用户
func (s *AdminService) UnlockUser(userID uint) error {
	result := s.db.Model(&models.User{}).Where("id = ?", userID).Update("status", models.UserStatusActive)
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return result.Error
}
