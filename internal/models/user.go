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
	FeishuID         *string        `gorm:"size:64;uniqueIndex" json:"feishu_id"`
	FeishuName       string         `gorm:"size:128" json:"feishu_name"`
	FeishuBotWebhook string         `gorm:"size:512" json:"feishu_bot_webhook"` // 飞书机器人 Webhook（notify_type=webhook 时使用）
	// 飞书通知方式：webhook | api | 空（不通知）
	FeishuNotifyType     string `gorm:"size:16" json:"feishu_notify_type"`
	FeishuAppID          string `gorm:"size:64" json:"-"`   // 不返回给前端
	FeishuAppSecret      string `gorm:"size:128" json:"-"`  // 不返回给前端
	FeishuReceiveIDType  string `gorm:"size:16" json:"feishu_receive_id_type"`  // chat_id | user_id | open_id
	FeishuReceiveID      string `gorm:"size:128" json:"feishu_receive_id"`
	Status           string         `gorm:"size:16;default:locked;not null" json:"status"`
	IsSuperAdmin bool           `gorm:"default:false" json:"is_super_admin"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}
