package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUser_FeishuBotWebhook(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))

	webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	u := User{
		Username:         "testuser",
		PasswordHash:     "hash",
		Status:           UserStatusActive,
		FeishuBotWebhook: webhook,
	}
	require.NoError(t, db.Create(&u).Error)

	var got User
	require.NoError(t, db.First(&got, u.ID).Error)
	require.Equal(t, webhook, got.FeishuBotWebhook)
}
