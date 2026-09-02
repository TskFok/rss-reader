package models

import "time"

const AppSettingSingletonID uint = 1

type AppSetting struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	PasswordLoginEnabled bool      `gorm:"not null;default:true" json:"password_login_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (AppSetting) TableName() string {
	return "app_settings"
}
