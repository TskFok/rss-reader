package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	JWT        JWTConfig        `yaml:"jwt"`
	SuperAdmin SuperAdminConfig `yaml:"super_admin"`
	Feishu     FeishuConfig     `yaml:"feishu"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port int `yaml:"port"`
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
	if v := os.Getenv("FEISHU_APP_ID"); v != "" {
		cfg.Feishu.AppID = v
	}
	if v := os.Getenv("FEISHU_APP_SECRET"); v != "" {
		cfg.Feishu.AppSecret = v
	}
	if v := os.Getenv("FEISHU_REDIRECT"); v != "" {
		cfg.Feishu.Redirect = v
	}
}
