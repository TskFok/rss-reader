package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupFeishuDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

func TestFeishuAuthService_LoginOrCreateByFeishu(t *testing.T) {
	db := setupFeishuDB(t)
	svc := NewFeishuAuthService(db)

	info := FeishuUserInfo{
		OpenID: "open_1",
		Name:   "张三",
		Email:  "zhangsan@example.com",
	}
	u, created, locked, err := svc.LoginOrCreateByFeishu(info)
	require.NoError(t, err)
	assert.True(t, created)
	assert.True(t, locked)
	require.NotNil(t, u.FeishuID)
	assert.Equal(t, "open_1", *u.FeishuID)
	assert.Equal(t, models.UserStatusLocked, u.Status)

	u2, created2, locked2, err := svc.LoginOrCreateByFeishu(info)
	require.NoError(t, err)
	assert.False(t, created2)
	assert.True(t, locked2)
	assert.Equal(t, u.ID, u2.ID)
}

func TestFeishuAuthService_BindFeishuToUser(t *testing.T) {
	db := setupFeishuDB(t)
	svc := NewFeishuAuthService(db)

	u1 := &models.User{Username: "u1", PasswordHash: "x", Status: models.UserStatusLocked}
	u2 := &models.User{Username: "u2", PasswordHash: "x", Status: models.UserStatusLocked}
	require.NoError(t, db.Create(u1).Error)
	require.NoError(t, db.Create(u2).Error)

	info := FeishuUserInfo{OpenID: "open_bind", Name: "张三"}
	require.NoError(t, svc.BindFeishuToUser(u1.ID, info))

	var got models.User
	require.NoError(t, db.First(&got, u1.ID).Error)
	require.NotNil(t, got.FeishuID)
	assert.Equal(t, "open_bind", *got.FeishuID)

	// 绑定到第二个用户应失败
	err := svc.BindFeishuToUser(u2.ID, info)
	assert.Error(t, err)
}

