package models

import (
	"time"

	"gorm.io/gorm"
)

// FeedCategory 订阅分类
type FeedCategory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index;uniqueIndex:idx_feed_category_user_name" json:"user_id"`
	Name      string         `gorm:"size:64;not null;uniqueIndex:idx_feed_category_user_name" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User  User   `gorm:"foreignKey:UserID" json:"-"`
	Feeds []Feed `gorm:"foreignKey:CategoryID" json:"-"`
}

func (FeedCategory) TableName() string {
	return "feed_categories"
}

