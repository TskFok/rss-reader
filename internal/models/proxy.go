package models

import (
	"time"

	"gorm.io/gorm"
)

// Proxy 代理配置
type Proxy struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Name      string         `gorm:"size:64" json:"name"`
	URL       string         `gorm:"size:512;not null" json:"url"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Proxy) TableName() string {
	return "proxies"
}
