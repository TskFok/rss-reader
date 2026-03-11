package models

import (
	"time"

	"gorm.io/gorm"
)

// ErrorLog 错误日志（落库）
type ErrorLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    *uint          `gorm:"index" json:"user_id"`
	Level     string         `gorm:"size:16" json:"level"` // error/panic/etc
	Message   string         `gorm:"type:text" json:"message"`
	Location  string         `gorm:"size:512" json:"location"` // file:line 或 METHOD PATH
	Method    string         `gorm:"size:16" json:"method"`
	Path      string         `gorm:"size:512" json:"path"`
	Status    int            `json:"status"`
	Stack     string         `gorm:"type:longtext" json:"stack"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ErrorLog) TableName() string {
	return "error_logs"
}

