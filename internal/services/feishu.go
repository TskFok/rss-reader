package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

// 使用 passport 体系接口（与 www.feishu.cn/passport.feishu.cn 授权页配套）
// 若使用 open.feishu.cn 的 token 接口会导致「飞书授权失败」（code 来自 passport，不兼容）
const (
	feishuTokenURL   = "https://passport.feishu.cn/suite/passport/oauth/token"
	feishuUserInfoURL = "https://passport.feishu.cn/suite/passport/oauth/userinfo"
)

// FeishuUserInfo 飞书返回的用户信息（简化）
type FeishuUserInfo struct {
	OpenID string
	Name   string
	Email  string
}

// FeishuAPI 抽象飞书 HTTP 调用，便于测试替换
type FeishuAPI interface {
	GetUserInfo(code string) (FeishuUserInfo, error)
}

type feishuHTTPClient struct {
	appID       string
	appSecret   string
	redirectURI string
	client      *http.Client
}

func NewFeishuHTTPClient(appID, appSecret, redirectURI string) FeishuAPI {
	return &feishuHTTPClient{
		appID:       appID,
		appSecret:   appSecret,
		redirectURI: redirectURI,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// passport token 接口直接返回 { access_token, token_type, expires_in, ... }，无 code/msg/data 包装
type feishuTokenData struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// passport 错误响应
type feishuTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// passport userinfo 直接返回用户对象
type feishuUserInfoRaw struct {
	OpenID   string `json:"open_id"`
	UnionID  string `json:"union_id"`
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Sub      string `json:"sub"` // 部分返回用 sub 作为 open_id
}

func (c *feishuHTTPClient) GetUserInfo(code string) (FeishuUserInfo, error) {
	if c.appID == "" || c.appSecret == "" {
		return FeishuUserInfo{}, errors.New("飞书应用未配置")
	}
	if c.redirectURI == "" {
		return FeishuUserInfo{}, errors.New("飞书 redirect_uri 未配置")
	}

	// 1. 用 code 换 access_token（passport 要求 application/x-www-form-urlencoded）
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.appID)
	form.Set("client_secret", c.appSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURI)

	req, err := http.NewRequest("POST", feishuTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("请求飞书 token 失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("读取 token 响应失败: %w", err)
	}

	var tokenData feishuTokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return FeishuUserInfo{}, fmt.Errorf("解析 token 响应失败: %w", err)
	}
	if tokenData.AccessToken == "" {
		var errResp feishuTokenError
		_ = json.Unmarshal(data, &errResp)
		msg := errResp.ErrorDescription
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = string(data)
		}
		return FeishuUserInfo{}, fmt.Errorf("飞书返回错误: %s", msg)
	}

	// 2. 用 access_token 拉取用户信息
	req2, err := http.NewRequest("GET", feishuUserInfoURL, nil)
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("创建 userinfo 请求失败: %w", err)
	}
	req2.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	resp2, err := c.client.Do(req2)
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("请求飞书 userinfo 失败: %w", err)
	}
	defer resp2.Body.Close()

	data2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return FeishuUserInfo{}, fmt.Errorf("读取 userinfo 响应失败: %w", err)
	}

	var raw feishuUserInfoRaw
	if err := json.Unmarshal(data2, &raw); err != nil {
		return FeishuUserInfo{}, fmt.Errorf("解析 userinfo 失败: %w", err)
	}
	openID := raw.OpenID
	if openID == "" && raw.Sub != "" {
		openID = raw.Sub
	}
	if openID == "" {
		return FeishuUserInfo{}, errors.New("飞书返回的用户信息中无 open_id")
	}
	return FeishuUserInfo{
		OpenID: openID,
		Name:   raw.Name,
		Email:  raw.Email,
	}, nil
}

// FeishuAuthService 负责基于飞书信息创建/绑定用户
type FeishuAuthService struct {
	db *gorm.DB
}

func NewFeishuAuthService(db *gorm.DB) *FeishuAuthService {
	return &FeishuAuthService{db: db}
}

// LoginOrCreateByFeishu 通过飞书信息登录或创建用户
// 返回：user, created(是否新建), locked(是否锁定)
func (s *FeishuAuthService) LoginOrCreateByFeishu(info FeishuUserInfo) (*models.User, bool, bool, error) {
	if info.OpenID == "" {
		return nil, false, false, errors.New("缺少 open_id")
	}
	var user models.User
	err := s.db.Where("feishu_id = ?", info.OpenID).First(&user).Error
	if err == nil {
		// 已存在
		return &user, false, user.Status != models.UserStatusActive, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, false, err
	}

	// 按邮箱前缀或名字生成用户名，防止冲突
	username := genFeishuUsername(s.db, info)

	u := &models.User{
		Username:     username,
		PasswordHash: "", // 飞书登录不使用密码
		Status:       models.UserStatusLocked,
		IsSuperAdmin: false,
		FeishuID:     &info.OpenID,
		FeishuName:   info.Name,
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, false, false, err
	}
	return u, true, true, nil
}

// BindFeishuToUser 将 Feishu 账号绑定到指定用户
func (s *FeishuAuthService) BindFeishuToUser(userID uint, info FeishuUserInfo) error {
	if info.OpenID == "" {
		return errors.New("缺少 open_id")
	}
	// 检查是否被其它用户占用
	var count int64
	if err := s.db.Model(&models.User{}).
		Where("feishu_id = ? AND id <> ?", info.OpenID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该飞书账号已绑定其他用户")
	}
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"feishu_id":   info.OpenID,
			"feishu_name": info.Name,
		}).Error
}

func genFeishuUsername(db *gorm.DB, info FeishuUserInfo) string {
	base := ""
	if info.Email != "" {
		for i := 0; i < len(info.Email); i++ {
			if info.Email[i] == '@' {
				base = info.Email[:i]
				break
			}
		}
	}
	if base == "" && info.Name != "" {
		base = info.Name
	}
	if base == "" {
		base = "feishu"
	}
	username := base
	var count int64
	i := 1
	for {
		if err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return base
		}
		if count == 0 {
			return username
		}
		i++
		username = fmt.Sprintf("%s%d", base, i)
	}
}

