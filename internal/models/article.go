package models

import (
	"time"

	"gorm.io/gorm"
)

// 文章 AI 异步处理状态（写入库后排队处理）
const (
	AIProcessPending = "pending"
	AIProcessDone    = "done"
	AIProcessFailed  = "failed"
)

// Article 文章模型
type Article struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	FeedID      uint           `gorm:"index;not null" json:"feed_id"`
	GUID        string         `gorm:"column:guid;type:char(64);not null" json:"guid"`
	GUIDRaw     string         `gorm:"column:guid_raw;type:longtext;not null" json:"guid_raw"`
	Title       string         `gorm:"size:1024" json:"title"`
	Link        string         `gorm:"size:2048" json:"link"`
	Content     string         `gorm:"type:longtext" json:"content"`
	// AI 异步处理结果（仅在新条目写入后触发）
	AIProcessStatus string `gorm:"size:16;default:'';column:ai_process_status" json:"ai_process_status"` // pending|done|failed；须显式 column，否则 GORM 会生成 a_iprocess_status
	// 字段名避免使用 *Error 结尾，否则部分 GORM/SQLite 迁移会忽略该列
	AILastError     string `gorm:"size:512;default:''" json:"ai_last_error"`
	AICategory           string `gorm:"size:256;default:''" json:"ai_category"`
	AICategoryTranslated string `gorm:"size:256;default:''" json:"ai_category_translated"`
	TitleTranslated      string `gorm:"size:1024;default:''" json:"title_translated"`
	ContentTranslated    string `gorm:"type:longtext" json:"content_translated"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Feed         Feed          `gorm:"foreignKey:FeedID" json:"-"`
	UserArticles []UserArticle `gorm:"foreignKey:ArticleID" json:"-"`
}

// TableName 表名
func (Article) TableName() string {
	return "articles"
}
