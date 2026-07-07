package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/config"
	"github.com/tskfok/rss-reader/internal/models"
	"github.com/tskfok/rss-reader/internal/services"
	"gorm.io/gorm"
)

type fakeFeishuAPI struct {
	userInfo services.FeishuUserInfo
	err      error
}

func (f fakeFeishuAPI) GetUserInfo(_ string) (services.FeishuUserInfo, error) {
	if f.err != nil {
		return services.FeishuUserInfo{}, f.err
	}
	return f.userInfo, nil
}

func setupFeishuExchangeHandler(t *testing.T, api services.FeishuAPI) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	authSvc := services.NewAuthService(db, "secret", 24, "")
	feishuAuthSvc := services.NewFeishuAuthService(db)
	h := NewFeishuHandler(&config.FeishuConfig{}, api, authSvc, feishuAuthSvc)

	r := gin.New()
	r.POST("/api/auth/feishu/exchange", h.Exchange)
	return r, db
}

func TestFeishuExchange_Success(t *testing.T) {
	openID := "open_123"
	api := fakeFeishuAPI{
		userInfo: services.FeishuUserInfo{
			OpenID: openID,
			Name:   "Alice",
		},
	}
	r, db := setupFeishuExchangeHandler(t, api)
	require.NoError(t, db.Create(&models.User{
		Username:     "alice",
		PasswordHash: "",
		Status:       models.UserStatusActive,
		FeishuID:     &openID,
		FeishuName:   "Alice",
	}).Error)

	body := []byte(`{"code":"ok-code","state":"login"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/feishu/exchange", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
	user, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice", user["username"])
}

func TestFeishuExchange_LockedUser(t *testing.T) {
	api := fakeFeishuAPI{
		userInfo: services.FeishuUserInfo{
			OpenID: "open_new",
			Name:   "New User",
		},
	}
	r, _ := setupFeishuExchangeHandler(t, api)

	body := []byte(`{"code":"ok-code","state":"login"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/feishu/exchange", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	msg, _ := resp["error"].(string)
	assert.Contains(t, msg, "锁定")
}

func TestFeishuExchange_BindStateRejected(t *testing.T) {
	api := fakeFeishuAPI{
		err: errors.New("should not be called"),
	}
	r, _ := setupFeishuExchangeHandler(t, api)

	body := []byte(`{"code":"ok-code","state":"bind:1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/feishu/exchange", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	msg, _ := resp["error"].(string)
	assert.Contains(t, msg, "bind state")
}
