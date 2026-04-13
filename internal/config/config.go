package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	JWT        JWTConfig        `yaml:"jwt"`
	SuperAdmin SuperAdminConfig `yaml:"super_admin"`
	Feishu     FeishuConfig     `yaml:"feishu"`
	Log        LogConfig        `yaml:"log"`
	AIBackfill AIBackfillConfig `yaml:"ai_backfill"`
}

// AIBackfillConfig 定时补漏：对开启 AI 但未成功分类/翻译的文章分批重试，避免集中打爆模型接口
type AIBackfillConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalMinutes int  `yaml:"interval_minutes"` // 执行周期，默认 5
	ClassifyBatch   int  `yaml:"classify_batch"`   // 每次最多补分类条数，默认 3
	TranslateBatch  int  `yaml:"translate_batch"`  // 每次最多补翻译条数，默认 3
	DelayMS         int  `yaml:"delay_ms_between_calls"` // 每条之间的间隔毫秒，默认 600
}

// LogConfig 日志配置
type LogConfig struct {
	// Level 日志等级：debug, info, warn, error。正式环境建议设为 error 以关闭非异常日志
	Level string `yaml:"level"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port  int  `yaml:"port"`
	Debug bool `yaml:"debug"` // 是否开启 Gin debug 模式（会输出 GIN-debug 路由等信息），正式环境建议关闭
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

// SuperAdminConfig 超级管理员配置
type SuperAdminConfig struct {
	Username string `yaml:"username"`
}

// FeishuConfig 飞书登录配置
type FeishuConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	Redirect  string `yaml:"redirect"`
}

// Load 从文件加载配置，环境变量可覆盖
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("GIN_DEBUG"); v != "" {
		cfg.Server.Debug = v == "1" || strings.EqualFold(v, "true") || v == "on"
	}
	if v := os.Getenv("FEISHU_APP_ID"); v != "" {
		cfg.Feishu.AppID = v
	}
	if v := os.Getenv("FEISHU_APP_SECRET"); v != "" {
		cfg.Feishu.AppSecret = v
	}
	if v := os.Getenv("FEISHU_REDIRECT"); v != "" {
		cfg.Feishu.Redirect = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
}
