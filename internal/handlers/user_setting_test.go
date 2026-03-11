package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/config"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/services"
	"gorm.io/gorm"
)

func setupUserSettingHandlers(t *testing.T, db *gorm.DB, userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	userSettingSvc := services.NewUserSettingService(db)
	feishuBot := services.NewFeishuBotService(&config.FeishuConfig{AppID: "test", AppSecret: "test"})
	h := NewUserSettingHandler(userSettingSvc, feishuBot)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	})
	r.GET("/users/me/settings", h.GetSettings)
	r.PUT("/users/me/settings", h.UpdateSettings)
	r.POST("/users/me/feishu-bot/test", h.TestFeishuBot)
	return r
}

func TestUserSettingHandler_GetSettings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:get?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	req := httptest.NewRequest(http.MethodGet, "/users/me/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp GetSettingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "", resp.FeishuBotWebhook)
}

func TestUserSettingHandler_GetSettings_WithWebhook(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:getwh?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/test"
	u := models.User{Username: "bob", PasswordHash: "h", Status: models.UserStatusActive, FeishuBotWebhook: webhook}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	req := httptest.NewRequest(http.MethodGet, "/users/me/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp GetSettingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, webhook, resp.FeishuBotWebhook)
}

func TestUserSettingHandler_UpdateSettings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/updated"
	body, _ := json.Marshal(UpdateSettingsRequest{FeishuBotWebhook: &webhook})
	req := httptest.NewRequest(http.MethodPut, "/users/me/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var user models.User
	require.NoError(t, db.Where("id = ?", u.ID).First(&user).Error)
	assert.Equal(t, webhook, user.FeishuBotWebhook)
}

func TestUserSettingHandler_TestFeishuBot_NoWebhook(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:testno?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	req := httptest.NewRequest(http.MethodPost, "/users/me/feishu-bot/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "请先配置")
}
