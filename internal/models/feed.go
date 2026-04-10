package models

import (
	"time"

	"gorm.io/gorm"
)

// Feed 订阅源模型
type Feed struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	UserID                uint           `gorm:"index;not null" json:"user_id"`
	CategoryID            *uint          `gorm:"index" json:"category_id"`
	ProxyID               *uint          `gorm:"index" json:"proxy_id"`
	// URL 长度 768 以兼容 MySQL InnoDB 索引键长度限制（utf8mb4 下 3072 字节）
	URL                   string         `gorm:"size:768;not null" json:"url"`
	Title                 string         `gorm:"size:512" json:"title"`
	UpdateIntervalMinutes int            `gorm:"default:60;not null" json:"update_interval_minutes"`
	ExpireDays            int            `gorm:"default:90;not null" json:"expire_days"` // 0=永不过期
	// AI：按订阅配置；新文章入库后由后台异步调用模型做分类/翻译（见 ArticleAIProcessor）
	AIModelID           *uint  `gorm:"index" json:"ai_model_id"`
	AIClassifyEnabled   bool   `gorm:"default:false;not null" json:"ai_classify_enabled"`
	AITranslateEnabled  bool   `gorm:"default:false;not null" json:"ai_translate_enabled"`
	AITargetLanguage    string `gorm:"size:32;default:''" json:"ai_target_language"` // 如 zh-CN、en，开启翻译时必填
	LastFetchedAt         *time.Time     `json:"last_fetched_at"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`

	User     User         `gorm:"foreignKey:UserID" json:"-"`
	Category *FeedCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Proxy    *Proxy       `gorm:"foreignKey:ProxyID" json:"proxy,omitempty"`
	Articles []Article    `gorm:"foreignKey:FeedID" json:"-"`
}

// TableName 表名
func (Feed) TableName() string {
	return "feeds"
}
