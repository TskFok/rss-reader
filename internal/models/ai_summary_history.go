package models

import (
	"time"

	"gorm.io/gorm"
)

// AISummaryHistory AI 总结历史记录
type AISummaryHistory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	AIModelID uint           `gorm:"not null;index" json:"ai_model_id"`

	// 查询条件（用于回放/理解本次总结的范围）
	FeedIDsJSON string `gorm:"type:text" json:"feed_ids_json"`
	StartTime   string `gorm:"size:32" json:"start_time"`
	EndTime     string `gorm:"size:32" json:"end_time"`
	Page        int    `gorm:"default:1" json:"page"`
	PageSize    int    `gorm:"default:20" json:"page_size"`
	Order       string `gorm:"size:8" json:"order"` // asc/desc

	ArticleCount int   `gorm:"default:0" json:"article_count"` // 本页参与总结的文章数
	Total        int64 `gorm:"default:0" json:"total"`         // 匹配条件的总文章数（分页前）

	Content string `gorm:"type:longtext" json:"content"`
	Error   string `gorm:"type:text" json:"error"`

	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	AIModel AIModel `gorm:"foreignKey:AIModelID" json:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"-"`
}

func (AISummaryHistory) TableName() string {
	return "ai_summary_histories"
}

