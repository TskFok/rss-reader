package services

import (
	"strings"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

// UserSettingService 用户设置服务
type UserSettingService struct {
	db *gorm.DB
}

// NewUserSettingService 创建用户设置服务
func NewUserSettingService(db *gorm.DB) *UserSettingService {
	return &UserSettingService{db: db}
}

// FeishuNotifyConfig 飞书通知配置
// 服务端 API 模式使用 config 的 app_id/app_secret，接收者为 users.feishu_id
type FeishuNotifyConfig struct {
	NotifyType string `json:"feishu_notify_type"`
	Webhook    string `json:"feishu_bot_webhook"`
	FeishuID   string `json:"feishu_id"` // 用户绑定的飞书 open_id，API 模式用于接收者
}

// GetFeishuNotifyConfig 获取当前用户的飞书通知配置
func (s *UserSettingService) GetFeishuNotifyConfig(userID uint) (*FeishuNotifyConfig, error) {
	var user models.User
	if err := s.db.Select("feishu_bot_webhook", "feishu_notify_type", "feishu_id").
		Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	cfg := &FeishuNotifyConfig{
		NotifyType: strings.TrimSpace(user.FeishuNotifyType),
		Webhook:    strings.TrimSpace(user.FeishuBotWebhook),
	}
	if user.FeishuID != nil {
		cfg.FeishuID = strings.TrimSpace(*user.FeishuID)
	}
	if cfg.NotifyType == "" && cfg.Webhook != "" {
		cfg.NotifyType = "webhook"
	}
	return cfg, nil
}

// UpdateFeishuNotifyConfig 更新飞书通知配置
func (s *UserSettingService) UpdateFeishuNotifyConfig(userID uint, cfg *FeishuNotifyConfig) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"feishu_notify_type": strings.TrimSpace(cfg.NotifyType),
		"feishu_bot_webhook": strings.TrimSpace(cfg.Webhook),
	}).Error
}

// GetFeishuNotifyConfigForSend 获取配置用于发送（含 feishu_id 用于 API 模式）
func (s *UserSettingService) GetFeishuNotifyConfigForSend(userID uint) (*FeishuNotifyConfig, error) {
	return s.GetFeishuNotifyConfig(userID)
}

// GetFeishuBotWebhook 获取当前用户的飞书机器人 Webhook（兼容旧逻辑）
func (s *UserSettingService) GetFeishuBotWebhook(userID uint) (string, error) {
	cfg, err := s.GetFeishuNotifyConfig(userID)
	if err != nil {
		return "", err
	}
	if cfg.NotifyType == "webhook" || (cfg.NotifyType == "" && cfg.Webhook != "") {
		return cfg.Webhook, nil
	}
	return "", nil
}

// UpdateFeishuBotWebhook 更新当前用户的飞书机器人 Webhook（兼容旧逻辑）
func (s *UserSettingService) UpdateFeishuBotWebhook(userID uint, webhook string) error {
	cfg, _ := s.GetFeishuNotifyConfig(userID)
	if cfg == nil {
		cfg = &FeishuNotifyConfig{}
	}
	cfg.Webhook = webhook
	if cfg.NotifyType == "" && webhook != "" {
		cfg.NotifyType = "webhook"
	}
	return s.UpdateFeishuNotifyConfig(userID, cfg)
}
