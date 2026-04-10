package models

import (
	"time"

	"gorm.io/gorm"
)

// AISummaryTemplate 用户可配置的 AI 总结 Prompt 模版（系统提示 + 用户侧说明前缀，其后由服务拼接文章列表）
type AISummaryTemplate struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"not null;index" json:"user_id"`
	Name             string         `gorm:"size:128;not null" json:"name"`
	SystemPrompt     string         `gorm:"type:text" json:"system_prompt"`           // 可选，作为 chat system 消息
	UserPromptPrefix string         `gorm:"type:longtext" json:"user_prompt_prefix"` // 可选；空则使用内置默认说明，再接「---」与文章正文
	SortOrder        int            `gorm:"default:0" json:"sort_order"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AISummaryTemplate) TableName() string {
	return "ai_summary_templates"
}
