package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/services"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthHandlers(t *testing.T) (*gin.Engine, *services.AuthService, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
	authSvc := services.NewAuthService(db, "secret", 24, "")
	h := NewAuthHandler(authSvc)
	r := gin.New()
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	return r, authSvc, db
}

func TestAuthHandler_Register(t *testing.T) {
	r, _, _ := setupAuthHandlers(t)

	body, _ := json.Marshal(services.RegisterRequest{Username: "alice", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "注册成功，请等待管理员解锁", resp["message"])
}

func TestAuthHandler_Register_Duplicate(t *testing.T) {
	r, _, _ := setupAuthHandlers(t)
	body, _ := json.Marshal(services.RegisterRequest{Username: "alice", Password: "password123"})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body)))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthHandler_Login(t *testing.T) {
	r, authSvc, db := setupAuthHandlers(t)
	_, _ = authSvc.Register(services.RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)

	body, _ := json.Marshal(services.LoginRequest{Username: "alice", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
}
