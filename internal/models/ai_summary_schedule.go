package models

import (
	"time"

	"gorm.io/gorm"
)

// AISummarySchedule 定时总结配置
type AISummarySchedule struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	AIModelID uint           `gorm:"not null;index" json:"ai_model_id"`

	FeedIDsJSON string `gorm:"type:text" json:"feed_ids_json"`

	// RunAt 每天执行时间（上海时区），格式 HH:MM
	RunAt string `gorm:"size:8;not null" json:"run_at"`

	PageSize int    `gorm:"default:20" json:"page_size"`
	Order    string `gorm:"size:8" json:"order"` // asc/desc
	Enabled  bool   `gorm:"default:true" json:"enabled"`

	LastRunAt *time.Time     `json:"last_run_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	AIModel AIModel `gorm:"foreignKey:AIModelID" json:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"-"`
}

func (AISummarySchedule) TableName() string {
	return "ai_summary_schedules"
}

