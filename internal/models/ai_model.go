package models

import (
	"time"

	"gorm.io/gorm"
)

// AIModel AI 模型配置
type AIModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Name      string         `gorm:"size:128" json:"name"`
	BaseURL   string         `gorm:"size:512;not null" json:"base_url"`
	APIKey    string         `gorm:"size:512" json:"-"` // 不返回给前端，仅存储
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (AIModel) TableName() string {
	return "ai_models"
}
