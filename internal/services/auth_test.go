package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
	return db
}

func TestAuthService_Register(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret", 24, "")

	user, err := svc.Register(RegisterRequest{Username: "alice", Password: "password123"})
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, models.UserStatusLocked, user.Status)
	assert.True(t, user.IsSuperAdmin, "首个用户应为超级管理员")

	_, err = svc.Register(RegisterRequest{Username: "alice", Password: "other"})
	assert.ErrorIs(t, err, ErrUserExists)
}

func TestAuthService_Login_Locked(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret", 24, "")
	_, _ = svc.Register(RegisterRequest{Username: "alice", Password: "password123"})

	_, err := svc.Login(LoginRequest{Username: "alice", Password: "password123"})
	assert.ErrorIs(t, err, ErrUserLocked)
}

func TestAuthService_Login_InvalidCreds(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret", 24, "")
	_, _ = svc.Register(RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)

	_, err := svc.Login(LoginRequest{Username: "alice", Password: "wrong"})
	assert.ErrorIs(t, err, ErrInvalidCreds)

	_, err = svc.Login(LoginRequest{Username: "bob", Password: "any"})
	assert.ErrorIs(t, err, ErrInvalidCreds)
}

func TestAuthService_Login_Success(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret", 24, "")
	_, _ = svc.Register(RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)

	res, err := svc.Login(LoginRequest{Username: "alice", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "alice", res.User.Username)
}

func TestAuthService_ValidateToken(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret", 24, "")
	_, _ = svc.Register(RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)
	res, _ := svc.Login(LoginRequest{Username: "alice", Password: "password123"})

	userID, err := svc.ValidateToken(res.Token)
	require.NoError(t, err)
	assert.Equal(t, uint(1), userID)

	_, err = svc.ValidateToken("invalid-token")
	assert.Error(t, err)
}
