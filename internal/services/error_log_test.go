package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupErrorLogDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ErrorLog{}))
	return db
}

func TestErrorLogService_Create(t *testing.T) {
	db := setupErrorLogDB(t)
	svc := NewErrorLogService(db)
	err := svc.Create(CreateErrorLogRequest{
		Level:    "error",
		Message:  "boom",
		Location: "x.go:1",
		Method:   "GET",
		Path:     "/api/test",
		Status:   500,
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.ErrorLog{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestErrorLogService_List(t *testing.T) {
	db := setupErrorLogDB(t)
	svc := NewErrorLogService(db)
	u := uint(1)
	require.NoError(t, svc.Create(CreateErrorLogRequest{UserID: &u, Message: "u1"}))
	require.NoError(t, svc.Create(CreateErrorLogRequest{UserID: nil, Message: "sys"}))
	require.NoError(t, svc.Create(CreateErrorLogRequest{UserID: func() *uint { v := uint(2); return &v }(), Message: "u2"}))

	items, total, err := svc.List(1, ListErrorLogsRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
}

