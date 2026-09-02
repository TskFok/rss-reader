package services

import (
	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

type AppSettingService struct {
	db *gorm.DB
}

func NewAppSettingService(db *gorm.DB) *AppSettingService {
	return &AppSettingService{db: db}
}

func (s *AppSettingService) GetPasswordLoginEnabled() (bool, error) {
	var setting models.AppSetting
	err := s.db.Where("id = ?", models.AppSettingSingletonID).
		Attrs(models.AppSetting{ID: models.AppSettingSingletonID, PasswordLoginEnabled: true}).
		FirstOrCreate(&setting).Error
	if err != nil {
		return false, err
	}
	return setting.PasswordLoginEnabled, nil
}

func (s *AppSettingService) SetPasswordLoginEnabled(enabled bool) error {
	if _, err := s.GetPasswordLoginEnabled(); err != nil {
		return err
	}
	return s.db.Model(&models.AppSetting{}).
		Where("id = ?", models.AppSettingSingletonID).
		Updates(map[string]interface{}{"password_login_enabled": enabled}).Error
}
