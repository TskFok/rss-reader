package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAdminDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
	return db
}

func TestAdminService_ListUsers(t *testing.T) {
	db := setupAdminDB(t)
	svc := NewAdminService(db)

	users, err := svc.ListUsers()
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestAdminService_UnlockUser_NotFound(t *testing.T) {
	db := setupAdminDB(t)
	svc := NewAdminService(db)

	err := svc.UnlockUser(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
