package models

import (
	"time"

	"gorm.io/gorm"
)

// UserStatus 用户状态
const (
	UserStatusLocked = "locked"
	UserStatusActive = "active"
)

// User 用户模型
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:255;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	FeishuID     *string        `gorm:"size:64;uniqueIndex" json:"feishu_id"`
	FeishuName   string         `gorm:"size:128" json:"feishu_name"`
	Status       string         `gorm:"size:16;default:locked;not null" json:"status"`
	IsSuperAdmin bool           `gorm:"default:false" json:"is_super_admin"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}
