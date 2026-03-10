package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := `
server:
  port: 8080
database:
  dsn: "user:pass@tcp(localhost:3306)/db"
jwt:
  secret: "test-secret"
  expire_hours: 24
super_admin:
  username: "admin"
feishu:
  app_id: "appid"
  app_secret: "secret"
  redirect: "http://localhost/api/auth/feishu/callback"
log:
  level: info
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "user:pass@tcp(localhost:3306)/db", cfg.Database.DSN)
	assert.Equal(t, "test-secret", cfg.JWT.Secret)
	assert.Equal(t, 24, cfg.JWT.ExpireHours)
	assert.Equal(t, "admin", cfg.SuperAdmin.Username)
	assert.Equal(t, "appid", cfg.Feishu.AppID)
	assert.Equal(t, "secret", cfg.Feishu.AppSecret)
	assert.Equal(t, "http://localhost/api/auth/feishu/callback", cfg.Feishu.Redirect)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.False(t, cfg.Server.Debug)
}

func TestLoad_ServerDebug(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := `
server:
  port: 8080
  debug: true
database:
  dsn: "user:pass@tcp(localhost:3306)/db"
jwt:
  secret: "test-secret"
  expire_hours: 24
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.True(t, cfg.Server.Debug)
}

func TestLoad_LogLevelDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := `
server:
  port: 8080
database:
  dsn: "user:pass@tcp(localhost:3306)/db"
jwt:
  secret: "test-secret"
  expire_hours: 24
super_admin:
  username: ""
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "info", cfg.Log.Level)
}

func TestLoad_LogLevelExplicit(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := `
server:
  port: 8080
database:
  dsn: "user:pass@tcp(localhost:3306)/db"
jwt:
  secret: "test-secret"
  expire_hours: 24
log:
  level: error
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "error", cfg.Log.Level)
}

func TestLoad_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := `
server:
  port: 8080
database:
  dsn: "default"
jwt:
  secret: "default-secret"
  expire_hours: 24
super_admin:
  username: ""
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	os.Setenv("DB_DSN", "env-dsn")
	os.Setenv("JWT_SECRET", "env-secret")
	os.Setenv("PORT", "9000")
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("GIN_DEBUG", "true")
	defer func() {
		os.Unsetenv("DB_DSN")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("GIN_DEBUG")
	}()

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "env-dsn", cfg.Database.DSN)
	assert.Equal(t, "env-secret", cfg.JWT.Secret)
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "error", cfg.Log.Level)
	assert.True(t, cfg.Server.Debug)
}
