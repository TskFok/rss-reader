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
	// BackupModelID 备用模型 ID（同一用户下的其他模型）。主模型调用失败时可尝试调用备用模型。
	BackupModelID *uint          `gorm:"index" json:"backup_model_id"`
	SortOrder int            `gorm:"default:0;not null" json:"sort_order"` // 拖动排序，越小越靠前
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (AIModel) TableName() string {
	return "ai_models"
}
