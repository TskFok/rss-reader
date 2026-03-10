package models

import (
	"time"

	"gorm.io/gorm"
)

// Article 文章模型
type Article struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	FeedID      uint           `gorm:"index;not null" json:"feed_id"`
	GUID        string         `gorm:"size:512;not null" json:"guid"`
	Title       string         `gorm:"size:1024" json:"title"`
	Link        string         `gorm:"size:1024" json:"link"`
	Content     string         `gorm:"type:longtext" json:"content"`
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
