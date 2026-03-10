package models

import (
	"time"

	"gorm.io/gorm"
)

// UserArticle 用户文章阅读状态与收藏
type UserArticle struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"uniqueIndex:idx_user_article;not null" json:"user_id"`
	ArticleID  uint           `gorm:"uniqueIndex:idx_user_article;not null" json:"article_id"`
	ReadStatus bool           `gorm:"default:false" json:"read_status"`
	Favorite   bool           `gorm:"default:false" json:"favorite"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	User    User    `gorm:"foreignKey:UserID" json:"-"`
	Article Article `gorm:"foreignKey:ArticleID" json:"-"`
}

// TableName 表名
func (UserArticle) TableName() string {
	return "user_articles"
}
