package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupAppSettingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AppSetting{}))
	return db
}

func TestAppSettingService_DefaultEnabled(t *testing.T) {
	db := setupAppSettingDB(t)
	svc := NewAppSettingService(db)

	enabled, err := svc.GetPasswordLoginEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestAppSettingService_SetAndGet(t *testing.T) {
	db := setupAppSettingDB(t)
	svc := NewAppSettingService(db)

	require.NoError(t, svc.SetPasswordLoginEnabled(false))
	enabled, err := svc.GetPasswordLoginEnabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	require.NoError(t, svc.SetPasswordLoginEnabled(true))
	enabled, err = svc.GetPasswordLoginEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}
