package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
	"github.com/tskfok/rss-reader/internal/services"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthHandlers(t *testing.T) (*gin.Engine, *services.AuthService, *services.AppSettingService, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}, &models.AppSetting{}))
	authSvc := services.NewAuthService(db, "secret", 24, "")
	appSettingSvc := services.NewAppSettingService(db)
	h := NewAuthHandler(authSvc, appSettingSvc)
	r := gin.New()
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	r.GET("/api/auth/login-options", h.LoginOptions)
	return r, authSvc, appSettingSvc, db
}

func TestAuthHandler_Register(t *testing.T) {
	r, _, _, _ := setupAuthHandlers(t)

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
	r, _, _, _ := setupAuthHandlers(t)
	body, _ := json.Marshal(services.RegisterRequest{Username: "alice", Password: "password123"})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body)))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthHandler_Login(t *testing.T) {
	r, authSvc, _, db := setupAuthHandlers(t)
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

func TestAuthHandler_Login_FormURLEncoded(t *testing.T) {
	r, authSvc, _, db := setupAuthHandlers(t)
	_, _ = authSvc.Register(services.RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)

	form := url.Values{}
	form.Set("username", "alice")
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
}

func TestAuthHandler_Login_MultipartFormData(t *testing.T) {
	r, authSvc, _, db := setupAuthHandlers(t)
	_, _ = authSvc.Register(services.RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("username", "alice"))
	require.NoError(t, writer.WriteField("password", "password123"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
}

func TestAuthHandler_LoginOptions_DefaultEnabled(t *testing.T) {
	r, _, _, _ := setupAuthHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["password_login_enabled"])
}

func TestAuthHandler_LoginAndRegister_Disabled(t *testing.T) {
	r, authSvc, appSettingSvc, db := setupAuthHandlers(t)
	_, _ = authSvc.Register(services.RegisterRequest{Username: "alice", Password: "password123"})
	db.Model(&models.User{}).Where("username = ?", "alice").Update("status", models.UserStatusActive)
	require.NoError(t, appSettingSvc.SetPasswordLoginEnabled(false))

	optReq := httptest.NewRequest(http.MethodGet, "/api/auth/login-options", nil)
	optW := httptest.NewRecorder()
	r.ServeHTTP(optW, optReq)
	assert.Equal(t, http.StatusOK, optW.Code)
	var opt map[string]interface{}
	require.NoError(t, json.Unmarshal(optW.Body.Bytes(), &opt))
	assert.Equal(t, false, opt["password_login_enabled"])

	loginBody, _ := json.Marshal(services.LoginRequest{Username: "alice", Password: "password123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	assert.Equal(t, http.StatusForbidden, loginW.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginW.Body.Bytes(), &loginResp))
	assert.Equal(t, "已关闭账号密码登录，请使用飞书登录", loginResp["error"])

	regBody, _ := json.Marshal(services.RegisterRequest{Username: "bob", Password: "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, regReq)
	assert.Equal(t, http.StatusForbidden, regW.Code)
	var regResp map[string]interface{}
	require.NoError(t, json.Unmarshal(regW.Body.Bytes(), &regResp))
	assert.Equal(t, "已关闭账号注册，请使用飞书登录", regResp["error"])
}
