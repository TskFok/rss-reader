# 关闭账号密码登录 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让超级管理员在「我的」页关闭全站账号密码登录与注册，关闭后所有人只能使用飞书登录。

**Architecture:** 用 `app_settings` 单行表保存 `password_login_enabled`（默认 `true`）。`AppSettingService` 负责读取/写入；`AuthHandler` 在 login/register 之前拒绝并提供公开 `GET /auth/login-options`；`UserSettingHandler` 把该字段并入现有用户设置接口，仅超管可写。Web 登录/注册页读公开接口决定入口，「我的」页仅超管显示开关。

**Tech Stack:** Go、Gin、GORM、React 19、TypeScript、Vitest、React Testing Library。

## Global Constraints

- 默认 `password_login_enabled=true`，与现有行为一致。
- 关闭后登录错误文案必须是「已关闭账号密码登录，请使用飞书登录」。
- 关闭后注册错误文案必须是「已关闭账号注册，请使用飞书登录」。
- 普通用户 PUT `password_login_enabled` 必须返回 `403`，错误为「需要超级管理员权限」。
- 关闭前不探测飞书是否可用。
- 已签发 JWT 继续有效；飞书登录不受影响。
- 不写入、不读取 `config.yaml` 中的该开关。
- 禁止在循环内查询数据库。
- 不得写入真实密钥或假密钥占位符。

## File Structure

- Create: `internal/models/app_setting.go` — 站点设置模型
- Create: `internal/services/app_setting.go` — 读取/写入 `password_login_enabled`
- Create: `internal/services/app_setting_test.go`
- Modify: `internal/database/database.go` — AutoMigrate `AppSetting`
- Modify: `internal/handlers/auth.go` — 注入设置服务、LoginOptions、关闭时拒绝 login/register
- Modify: `internal/handlers/auth_test.go`
- Modify: `internal/handlers/user_setting.go` — GET/PUT 增加字段，超管写入限制
- Modify: `internal/services/user_setting.go` — `IsSuperAdmin`
- Modify: `internal/handlers/user_setting_test.go`
- Modify: `cmd/server/main.go` — 接线
- Modify: `web/src/api/client.ts` — `getLoginOptions` 与 `UserSettings.password_login_enabled`
- Modify: `web/src/pages/Me.tsx` / `Me.test.tsx`
- Modify: `web/src/pages/Login.tsx` / `Login.test.tsx`
- Modify: `web/src/pages/Register.tsx` / `Register.test.tsx`
- Modify: `docs/openapi.yaml`、`docs/ANDROID_REST_API.md`

---

### Task 1: 站点设置模型与服务

**Files:**
- Create: `internal/models/app_setting.go`
- Create: `internal/services/app_setting.go`
- Create: `internal/services/app_setting_test.go`
- Modify: `internal/database/database.go`

**Interfaces:**
- Produces: `models.AppSetting`（`ID uint`、`PasswordLoginEnabled bool`）、`models.AppSettingSingletonID = 1`、`NewAppSettingService(db *gorm.DB) *AppSettingService`、`(*AppSettingService) GetPasswordLoginEnabled() (bool, error)`、`(*AppSettingService) SetPasswordLoginEnabled(enabled bool) error`
- Consumes: GORM `*gorm.DB`

- [ ] **Step 1: 写入失败测试**

创建 `internal/services/app_setting_test.go`：

```go
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
```

- [ ] **Step 2: 运行测试，确认因符号不存在而失败**

Run: `go test ./internal/services -run 'TestAppSettingService_' -count=1`

Expected: FAIL，编译错误指向 `models.AppSetting` 或 `NewAppSettingService` 未定义。

- [ ] **Step 3: 编写最小实现**

`internal/models/app_setting.go`：

```go
package models

import "time"

const AppSettingSingletonID uint = 1

type AppSetting struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	PasswordLoginEnabled bool      `gorm:"not null;default:true" json:"password_login_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (AppSetting) TableName() string {
	return "app_settings"
}
```

`internal/services/app_setting.go`：

```go
package services

import (
	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

type AppSettingService struct {
	db *gorm.DB
}

func NewAppSettingService(db *gorm.DB) *AppSettingService {
	return &AppSettingService{db: db}
}

func (s *AppSettingService) GetPasswordLoginEnabled() (bool, error) {
	var setting models.AppSetting
	err := s.db.Where("id = ?", models.AppSettingSingletonID).
		Attrs(models.AppSetting{ID: models.AppSettingSingletonID, PasswordLoginEnabled: true}).
		FirstOrCreate(&setting).Error
	if err != nil {
		return false, err
	}
	return setting.PasswordLoginEnabled, nil
}

func (s *AppSettingService) SetPasswordLoginEnabled(enabled bool) error {
	if _, err := s.GetPasswordLoginEnabled(); err != nil {
		return err
	}
	return s.db.Model(&models.AppSetting{}).
		Where("id = ?", models.AppSettingSingletonID).
		Updates(map[string]interface{}{"password_login_enabled": enabled}).Error
}
```

`SetPasswordLoginEnabled` 必须用 `map[string]interface{}` 更新，不能用结构体 `Update`，否则 `false` 会被 GORM 当成零值跳过。

在 `internal/database/database.go` 的 `AutoMigrate` 参数列表末尾增加 `&models.AppSetting{}`。

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/services -run 'TestAppSettingService_' -count=1`

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/models/app_setting.go internal/services/app_setting.go internal/services/app_setting_test.go internal/database/database.go
git commit -m "$(cat <<'EOF'
功能：站点设置表保存账号密码登录开关

EOF
)"
```

---

### Task 2: 公开 login-options，关闭时拒绝 login/register

**Files:**
- Modify: `internal/handlers/auth.go`
- Modify: `internal/handlers/auth_test.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `NewAppSettingService`、`GetPasswordLoginEnabled()`
- Produces: `NewAuthHandler(authSvc *services.AuthService, appSettingSvc *services.AppSettingService) *AuthHandler`、`(*AuthHandler) LoginOptions(c *gin.Context)`；`POST /api/auth/login` 与 `POST /api/auth/register` 在关闭时返回 403

- [ ] **Step 1: 写入失败测试**

在 `internal/handlers/auth_test.go` 中把 `setupAuthHandlers` 改为同时迁移 `AppSetting` 并注入设置服务，新增测试：

```go
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
```

把本文件里所有 `r, _, _ := setupAuthHandlers(t)` 改为 `r, _, _, _ := setupAuthHandlers(t)`；把 `r, authSvc, db := setupAuthHandlers(t)` 改为 `r, authSvc, _, db := setupAuthHandlers(t)`。

追加：

```go
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/handlers -run 'TestAuthHandler_' -count=1`

Expected: FAIL，`NewAuthHandler` 参数数量不匹配，且没有 `LoginOptions`。

- [ ] **Step 3: 编写最小实现**

`internal/handlers/auth.go`：

```go
type AuthHandler struct {
	authSvc        *services.AuthService
	appSettingSvc  *services.AppSettingService
}

func NewAuthHandler(authSvc *services.AuthService, appSettingSvc *services.AppSettingService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, appSettingSvc: appSettingSvc}
}

func (h *AuthHandler) LoginOptions(c *gin.Context) {
	enabled, err := h.appSettingSvc.GetPasswordLoginEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取登录方式失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"password_login_enabled": enabled})
}
```

在 `Register` 开头、绑定请求体之前：

```go
enabled, err := h.appSettingSvc.GetPasswordLoginEnabled()
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
	return
}
if !enabled {
	c.JSON(http.StatusForbidden, gin.H{"error": "已关闭账号注册，请使用飞书登录"})
	return
}
```

在 `Login` 开头、`ShouldBind` 之前：

```go
enabled, err := h.appSettingSvc.GetPasswordLoginEnabled()
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
	return
}
if !enabled {
	c.JSON(http.StatusForbidden, gin.H{"error": "已关闭账号密码登录，请使用飞书登录"})
	return
}
```

`cmd/server/main.go` 在创建 `userSettingSvc` 附近增加 `appSettingSvc := services.NewAppSettingService(db)`，并把 auth 路由改成：

```go
authHandler := handlers.NewAuthHandler(authSvc, appSettingSvc)
api.POST("/auth/register", authHandler.Register)
api.POST("/auth/login", authHandler.Login)
api.GET("/auth/login-options", authHandler.LoginOptions)
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/handlers -run 'TestAuthHandler_' -count=1`

Expected: PASS，含原有 login/register 成功用例与新增关闭用例。

- [ ] **Step 5: 提交**

```bash
git add internal/handlers/auth.go internal/handlers/auth_test.go cmd/server/main.go
git commit -m "$(cat <<'EOF'
功能：关闭账号密码登录后拒绝登录与注册

EOF
)"
```

---

### Task 3: 用户设置接口读写密码登录开关

**Files:**
- Modify: `internal/services/user_setting.go`
- Modify: `internal/handlers/user_setting.go`
- Modify: `internal/handlers/user_setting_test.go`
- Modify: `cmd/server/main.go`（`NewUserSettingHandler` 增加 `appSettingSvc`）

**Interfaces:**
- Consumes: `AppSettingService.GetPasswordLoginEnabled`、`SetPasswordLoginEnabled`
- Produces: `(*UserSettingService) IsSuperAdmin(userID uint) (bool, error)`；`GetSettingsResponse.PasswordLoginEnabled bool`；`UpdateSettingsRequest.PasswordLoginEnabled *bool`；`NewUserSettingHandler(userSettingSvc *services.UserSettingService, feishuBot services.FeishuBotClient, appSettingSvc *services.AppSettingService)`

- [ ] **Step 1: 写入失败测试**

更新 `setupUserSettingHandlers`：AutoMigrate `&models.AppSetting{}`，`NewUserSettingHandler` 传入 `services.NewAppSettingService(db)`。

追加测试（创建用户时设置 `IsSuperAdmin`）：

```go
func TestUserSettingHandler_GetSettings_IncludesPasswordLogin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:getpwd?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AppSetting{}))
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	req := httptest.NewRequest(http.MethodGet, "/users/me/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp GetSettingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.PasswordLoginEnabled)
}

func TestUserSettingHandler_UpdatePasswordLogin_SuperAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:updadmin?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AppSetting{}))
	u := models.User{Username: "admin", PasswordHash: "h", Status: models.UserStatusActive, IsSuperAdmin: true}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	enabled := false
	body, _ := json.Marshal(UpdateSettingsRequest{PasswordLoginEnabled: &enabled})
	req := httptest.NewRequest(http.MethodPut, "/users/me/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/users/me/settings", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	var resp GetSettingsResponse
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &resp))
	assert.False(t, resp.PasswordLoginEnabled)
}

func TestUserSettingHandler_UpdatePasswordLogin_Forbidden(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:upduser?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AppSetting{}))
	u := models.User{Username: "alice", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	r := setupUserSettingHandlers(t, db, u.ID)

	enabled := false
	webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/keep"
	body, _ := json.Marshal(UpdateSettingsRequest{PasswordLoginEnabled: &enabled, FeishuBotWebhook: &webhook})
	req := httptest.NewRequest(http.MethodPut, "/users/me/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "需要超级管理员权限", resp["error"])

	var user models.User
	require.NoError(t, db.First(&user, u.ID).Error)
	assert.Equal(t, "", user.FeishuBotWebhook)
}
```

保留 `TestUserSettingHandler_UpdateSettings`：普通用户只更新 webhook、不带 `password_login_enabled` 时仍应 200。

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/handlers -run 'TestUserSettingHandler_' -count=1`

Expected: FAIL，`NewUserSettingHandler` 参数或 `PasswordLoginEnabled` 字段不存在。

- [ ] **Step 3: 编写最小实现**

在 `internal/services/user_setting.go` 增加：

```go
func (s *UserSettingService) IsSuperAdmin(userID uint) (bool, error) {
	var user models.User
	if err := s.db.Select("is_super_admin").First(&user, userID).Error; err != nil {
		return false, err
	}
	return user.IsSuperAdmin, nil
}
```

`internal/handlers/user_setting.go`：

```go
type UserSettingHandler struct {
	userSettingSvc *services.UserSettingService
	feishuBot      services.FeishuBotClient
	appSettingSvc  *services.AppSettingService
}

func NewUserSettingHandler(userSettingSvc *services.UserSettingService, feishuBot services.FeishuBotClient, appSettingSvc *services.AppSettingService) *UserSettingHandler {
	return &UserSettingHandler{
		userSettingSvc: userSettingSvc,
		feishuBot:      feishuBot,
		appSettingSvc:  appSettingSvc,
	}
}

type GetSettingsResponse struct {
	FeishuNotifyType       string `json:"feishu_notify_type"`
	FeishuBotWebhook       string `json:"feishu_bot_webhook"`
	FeishuID               string `json:"feishu_id"`
	PasswordLoginEnabled   bool   `json:"password_login_enabled"`
}

type UpdateSettingsRequest struct {
	FeishuNotifyType     *string `json:"feishu_notify_type"`
	FeishuBotWebhook     *string `json:"feishu_bot_webhook"`
	PasswordLoginEnabled *bool   `json:"password_login_enabled"`
}
```

`GetSettings` 在返回前调用 `h.appSettingSvc.GetPasswordLoginEnabled()`，失败则 500；成功则写入 `PasswordLoginEnabled`。

`UpdateSettings` 在更新飞书配置之前：

```go
if req.PasswordLoginEnabled != nil {
	isAdmin, err := h.userSettingSvc.IsSuperAdmin(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设置失败"})
		return
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要超级管理员权限"})
		return
	}
	if err := h.appSettingSvc.SetPasswordLoginEnabled(*req.PasswordLoginEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设置失败"})
		return
	}
}
```

`cmd/server/main.go` 中 `NewUserSettingHandler(userSettingSvc, feishuBotSvc)` 改为 `NewUserSettingHandler(userSettingSvc, feishuBotSvc, appSettingSvc)`。

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/handlers -run 'TestUserSettingHandler_' -count=1`

Expected: PASS

再跑：`go test ./internal/handlers ./internal/services ./internal/database -count=1`

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/services/user_setting.go internal/handlers/user_setting.go internal/handlers/user_setting_test.go cmd/server/main.go
git commit -m "$(cat <<'EOF'
功能：超级管理员可通过设置接口关闭账号密码登录

EOF
)"
```

---

### Task 4: 「我的」页开关

**Files:**
- Modify: `web/src/api/client.ts`
- Modify: `web/src/pages/Me.tsx`
- Modify: `web/src/pages/Me.test.tsx`

**Interfaces:**
- Consumes: `GET /api/users/me/settings`、`PUT /api/users/me/settings` 的 `password_login_enabled`
- Produces: `authApi.getLoginOptions(): Promise<AxiosResponse<{ password_login_enabled: boolean }>>`（本任务可先加到 client，登录页下一任务使用）；`UserSettings.password_login_enabled?: boolean`

- [ ] **Step 1: 写入失败测试**

在 `web/src/api/client.ts` 的 `authApi` 增加：

```ts
getLoginOptions: () => client.get<{ password_login_enabled: boolean }>('/auth/login-options'),
```

`UserSettings` 增加 `password_login_enabled: boolean`。

`web/src/pages/Me.test.tsx` 增加 mock 与用例。在文件顶部：

```ts
import { waitFor } from '@testing-library/react';
import { ToastProvider } from '../contexts/ToastContext';
import { userSettingsApi } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    userSettingsApi: {
      ...actual.userSettingsApi,
      get: vi.fn().mockResolvedValue({
        data: {
          feishu_notify_type: '',
          feishu_bot_webhook: '',
          feishu_id: '',
          password_login_enabled: true,
        },
      }),
      update: vi.fn().mockResolvedValue({ data: { message: '保存成功' } }),
    },
  };
});
```

把 `renderMe` 改为接受 `isSuperAdmin = false`，用户 JSON 使用该值；外层包 `ToastProvider`。

现有用例断言：`expect(screen.queryByRole('heading', { name: '账号密码登录' })).not.toBeInTheDocument();`

追加：

```ts
test('超级管理员可确认关闭账号密码登录', async () => {
  const user = userEvent.setup();
  vi.mocked(userSettingsApi.update).mockClear();
  renderMe(true);

  const toggle = await screen.findByRole('button', { name: '已开启' });
  await user.click(toggle);
  expect(screen.getByRole('dialog')).toBeInTheDocument();
  expect(screen.getByText(/同时关闭注册/)).toBeInTheDocument();
  expect(screen.getByText(/不会检查飞书/)).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: '确认关闭' }));
  await waitFor(() => {
    expect(userSettingsApi.update).toHaveBeenCalledWith({ password_login_enabled: false });
  });
  expect(await screen.findByRole('button', { name: '已关闭' })).toBeInTheDocument();
});
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `npm test -- src/pages/Me.test.tsx`（在 `web/` 目录）

Expected: FAIL，页面没有「账号密码登录」开关。

- [ ] **Step 3: 编写最小实现**

`Me.tsx`：仅当 `user?.is_super_admin` 时渲染该行，放在「登录状态」上方。挂载后 `userSettingsApi.get()` 读取 `password_login_enabled`。按钮文案为「已开启」/「已关闭」，使用 `className="nice-admin-header-theme"`。

点击「已开启」打开现有 `Modal`，标题「关闭账号密码登录」，正文说明会同时关闭注册、且不会检查飞书是否可用；底部「取消」与「确认关闭」。确认后 `userSettingsApi.update({ password_login_enabled: false })`。点击「已关闭」直接 `update({ password_login_enabled: true })`，无需确认。保存中按钮 `disabled`。失败时用 `useToast({ message, variant: 'error' })`，不改本地开关状态。

普通用户不请求该开关也可以，但超管必须 GET。为避免普通用户多余请求，只在 `is_super_admin` 时 GET。

- [ ] **Step 4: 运行测试，确认通过**

Run: `npm test -- src/pages/Me.test.tsx`

Expected: PASS，含原有退出/主题用例与超管关闭用例。

- [ ] **Step 5: 提交**

```bash
git add web/src/api/client.ts web/src/pages/Me.tsx web/src/pages/Me.test.tsx
git commit -m "$(cat <<'EOF'
功能：我的页提供关闭账号密码登录开关

EOF
)"
```

---

### Task 5: 登录页与注册页按开关隐藏入口

**Files:**
- Modify: `web/src/pages/Login.tsx`
- Modify: `web/src/pages/Login.test.tsx`
- Modify: `web/src/pages/Register.tsx`
- Modify: `web/src/pages/Register.test.tsx`

**Interfaces:**
- Consumes: `authApi.getLoginOptions()`
- Produces: 关闭时登录页仅飞书入口；注册页重定向到 `/login` 并带 state 文案「已关闭账号注册，请使用飞书登录」

- [ ] **Step 1: 写入失败测试**

`Login.test.tsx`：现有用例不 mock `getLoginOptions` 时，请求失败必须仍显示「账号密码登录」（拉取失败视为开启）。新增 mock 用例：

```ts
import { waitFor } from '@testing-library/react';
import { authApi } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    authApi: {
      ...actual.authApi,
      getLoginOptions: vi.fn().mockResolvedValue({ data: { password_login_enabled: true } }),
      getFeishuLoginUrl: vi.fn().mockResolvedValue({ data: { url: '/api/auth/feishu/login', goto: 'https://example.com' } }),
    },
  };
});
```

现有玻璃态用例改为 `await screen.findByRole('button', { name: '账号密码登录' })`，避免与异步 fetch 竞态。

追加：

```ts
test('关闭账号密码登录后只保留飞书登录', async () => {
  vi.mocked(authApi.getLoginOptions).mockResolvedValueOnce({
    data: { password_login_enabled: false },
  } as Awaited<ReturnType<typeof authApi.getLoginOptions>>);

  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => {
      store.set(k, String(v));
    },
    removeItem: (k: string) => {
      store.delete(k);
    },
  };

  render(
    <MemoryRouter initialEntries={['/login']}>
      <ThemeProvider>
        <AuthProvider>
          <Login />
        </AuthProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await waitFor(() => {
    expect(screen.queryByRole('button', { name: '账号密码登录' })).not.toBeInTheDocument();
  });
  expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument();
  expect(screen.queryByText('还没有账号？')).not.toBeInTheDocument();
});
```

`Register.test.tsx`：拉取失败时仍显示注册表单。新增：

```ts
vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    authApi: {
      ...actual.authApi,
      getLoginOptions: vi.fn().mockResolvedValue({ data: { password_login_enabled: true } }),
    },
  };
});

test('关闭账号注册后跳转登录页', async () => {
  vi.mocked(authApi.getLoginOptions).mockResolvedValueOnce({
    data: { password_login_enabled: false },
  } as Awaited<ReturnType<typeof authApi.getLoginOptions>>);

  render(
    <MemoryRouter initialEntries={['/register']}>
      <ThemeProvider>
        <Routes>
          <Route path="/register" element={<Register />} />
          <Route path="/login" element={<div>登录页</div>} />
        </Routes>
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(await screen.findByText('登录页')).toBeInTheDocument();
});
```

现有注册表单用例改为 `await screen.findByRole('heading', { name: '注册' })`。

- [ ] **Step 2: 运行测试，确认失败**

Run: `npm test -- src/pages/Login.test.tsx src/pages/Register.test.tsx`

Expected: FAIL，关闭开关后仍能看到密码登录 Tab / 注册表单。

- [ ] **Step 3: 编写最小实现**

`Login.tsx`：挂载时调用 `authApi.getLoginOptions()`。`password_login_enabled === false` 时 `setMode('feishu')`，不渲染账号密码 Tab，不渲染注册链接。请求失败则保持默认 `mode='password'` 并显示现有入口。

`Register.tsx`：挂载时调用 `authApi.getLoginOptions()`。为 `false` 时 `navigate('/login', { replace: true, state: { message: '已关闭账号注册，请使用飞书登录' } })`。失败则继续渲染注册表单。

- [ ] **Step 4: 运行测试，确认通过**

Run: `npm test -- src/pages/Login.test.tsx src/pages/Register.test.tsx src/pages/Me.test.tsx`

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add web/src/pages/Login.tsx web/src/pages/Login.test.tsx web/src/pages/Register.tsx web/src/pages/Register.test.tsx
git commit -m "$(cat <<'EOF'
功能：登录与注册页按站点开关隐藏密码入口

EOF
)"
```

---

### Task 6: API 文档

**Files:**
- Modify: `docs/openapi.yaml`
- Modify: `docs/ANDROID_REST_API.md`

**Interfaces:**
- Consumes: 已实现的 `GET /api/auth/login-options`、login/register 403、settings 字段
- Produces: 与实现对齐的文档

- [ ] **Step 1: 更新 OpenAPI**

在 `paths` 中 ` /auth/login` 之前插入：

```yaml
  /auth/login-options:
    get:
      tags: [auth]
      summary: 查询是否允许账号密码登录（公开）
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  password_login_enabled:
                    type: boolean
```

`/auth/login` 的 `403` description 改为：账号被锁定，或站点已关闭账号密码登录。

新增：

```yaml
  /auth/register:
    post:
      tags: [auth]
      summary: 用户名密码注册
      responses:
        '201':
          description: 注册成功，等待管理员解锁
        '403':
          description: 站点已关闭账号注册
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorBody'
        '409':
          $ref: '#/components/responses/Error'

  /users/me/settings:
    get:
      tags: [auth]
      summary: 当前用户设置
      security: [{ bearerAuth: [] }]
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  feishu_notify_type: { type: string }
                  feishu_bot_webhook: { type: string }
                  feishu_id: { type: string }
                  password_login_enabled: { type: boolean }
    put:
      tags: [auth]
      summary: 更新当前用户设置
      description: |
        `password_login_enabled` 仅超级管理员可写；普通用户提交该字段返回 403「需要超级管理员权限」，且不会更新其它字段。
      security: [{ bearerAuth: [] }]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                feishu_notify_type: { type: string }
                feishu_bot_webhook: { type: string }
                password_login_enabled: { type: boolean }
      responses:
        '200':
          description: 保存成功
        '403':
          description: 普通用户试图修改 password_login_enabled
```

- [ ] **Step 2: 更新 Android 文档**

在 `docs/ANDROID_REST_API.md` 第 1 节「常见错误」改为：`401` 用户名或密码错误；`403` 账号被锁定，或站点已关闭账号密码登录（错误为「已关闭账号密码登录，请使用飞书登录」）。

在第 1 节后增加 1.1：

```markdown
### 1.1 查询登录方式（公开）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/auth/login-options` |
| 认证 | 不需要 |

**成功 `200`**：`{ "password_login_enabled": true }`

当 `password_login_enabled` 为 `false` 时，客户端应只展示飞书登录，不要调用 `POST /api/auth/login` 或注册接口。
```

在第 1.1 节后增加注册说明：

```markdown
### 1.2 注册（用户名 / 密码）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/auth/register` |
| 认证 | 不需要 |
| 请求体 | `{ "username": "string", "password": "string" }` |

**成功 `201`**：账号默认锁定。

**常见错误**：`409` 用户名已存在；`403` 站点已关闭账号注册（错误为「已关闭账号注册，请使用飞书登录」）。
```

- [ ] **Step 3: 全量回归**

Run: `go test ./...`

Expected: PASS

Run: `npm test`（在 `web/` 目录）

Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add docs/openapi.yaml docs/ANDROID_REST_API.md
git commit -m "$(cat <<'EOF'
文档：补充关闭账号密码登录的 API 说明

EOF
)"
```

---

## Spec coverage

| Spec 要求 | Task |
|-----------|------|
| `app_settings` 单行、默认 true、不写 yaml | 1 |
| `GET /api/auth/login-options` | 2 |
| 关闭后 login/register 403 与指定文案 | 2 |
| 设置接口读写、仅超管可写、普通用户 403 | 3 |
| 「我的」开关、确认关闭、失败保持原状态 | 4 |
| 登录页隐藏密码 Tab 与注册链接 | 5 |
| 注册页关闭时跳转登录 | 5 |
| OpenAPI 与 Android 文档 | 6 |
| 已有 JWT / 飞书登录不受影响 | 2（未改飞书与 JWT 校验） |
