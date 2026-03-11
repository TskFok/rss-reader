package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupUserSettingDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

func TestUserSettingService_GetFeishuBotWebhook(t *testing.T) {
	db := setupUserSettingDB(t)
	svc := NewUserSettingService(db)
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)

	webhook, err := svc.GetFeishuBotWebhook(u.ID)
	require.NoError(t, err)
	assert.Equal(t, "", webhook)
}

func TestUserSettingService_GetFeishuBotWebhook_WithValue(t *testing.T) {
	db := setupUserSettingDB(t)
	svc := NewUserSettingService(db)
	wh := "https://open.feishu.cn/open-apis/bot/v2/hook/test"
	u := models.User{Username: "bob", PasswordHash: "h", Status: models.UserStatusActive, FeishuBotWebhook: wh}
	require.NoError(t, db.Create(&u).Error)

	webhook, err := svc.GetFeishuBotWebhook(u.ID)
	require.NoError(t, err)
	assert.Equal(t, wh, webhook)
}

func TestUserSettingService_UpdateFeishuBotWebhook(t *testing.T) {
	db := setupUserSettingDB(t)
	svc := NewUserSettingService(db)
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)

	wh := "https://open.feishu.cn/open-apis/bot/v2/hook/updated"
	require.NoError(t, svc.UpdateFeishuBotWebhook(u.ID, wh))

	var user models.User
	require.NoError(t, db.Where("id = ?", u.ID).First(&user).Error)
	assert.Equal(t, wh, user.FeishuBotWebhook)
}

func TestUserSettingService_GetFeishuNotifyConfig_APIMode(t *testing.T) {
	db := setupUserSettingDB(t)
	svc := NewUserSettingService(db)
	feishuID := "open_xxx"
	u := models.User{
		Username:         "api-user",
		PasswordHash:     "h",
		Status:           models.UserStatusActive,
		FeishuNotifyType: "api",
		FeishuID:         &feishuID,
	}
	require.NoError(t, db.Create(&u).Error)

	cfg, err := svc.GetFeishuNotifyConfig(u.ID)
	require.NoError(t, err)
	assert.Equal(t, "api", cfg.NotifyType)
	assert.Equal(t, "open_xxx", cfg.FeishuID)
}

func TestUserSettingService_UpdateFeishuNotifyConfig_APIMode(t *testing.T) {
	db := setupUserSettingDB(t)
	svc := NewUserSettingService(db)
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)

	cfg := &FeishuNotifyConfig{NotifyType: "api", Webhook: ""}
	require.NoError(t, svc.UpdateFeishuNotifyConfig(u.ID, cfg))

	var user models.User
	require.NoError(t, db.Where("id = ?", u.ID).First(&user).Error)
	assert.Equal(t, "api", user.FeishuNotifyType)
}
