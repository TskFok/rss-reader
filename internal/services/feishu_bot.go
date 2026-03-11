package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ushopal/rss-reader/internal/config"
	"github.com/ushopal/rss-reader/internal/logger"
)

const (
	feishuBotTimeout       = 10 * time.Second
	feishuBotRetries       = 2
	feishuTokenRefreshBef  = 5 * time.Minute // 过期前 5 分钟刷新
	feishuTokenCacheMaxAge = 2 * time.Hour
)

// FeishuBotClient 抽象飞书机器人 HTTP 调用，便于测试替换
type FeishuBotClient interface {
	SendText(webhook string, title string, content string) error
	SendViaAPI(appID, appSecret, receiveIDType, receiveID, title, content string) error
	// SendToUserByOpenID 使用配置的 app_id/app_secret 向指定 open_id 用户发送消息
	SendToUserByOpenID(openID string, title string, content string) error
}

// FeishuBotService 封装发送文本消息到飞书机器人的逻辑（Webhook + 服务端 API）
type FeishuBotService struct {
	client      *http.Client
	tokenMu     sync.RWMutex
	tokenCache  map[string]*tokenEntry
	feishuCfg   *config.FeishuConfig
}

type tokenEntry struct {
	token   string
	expires time.Time
}

// NewFeishuBotService 创建 FeishuBotService，feishuCfg 用于服务端 API 模式的 app_id/app_secret
func NewFeishuBotService(feishuCfg *config.FeishuConfig) *FeishuBotService {
	return &FeishuBotService{
		client:     &http.Client{Timeout: feishuBotTimeout},
		tokenCache: make(map[string]*tokenEntry),
		feishuCfg:  feishuCfg,
	}
}

// feishuBotRequest 飞书机器人请求体
type feishuBotRequest struct {
	MsgType string             `json:"msg_type"`
	Content feishuBotTextContent `json:"content"`
}

type feishuBotTextContent struct {
	Text string `json:"text"`
}

// feishuBotResponse 飞书机器人响应体
type feishuBotResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendText 发送文本消息到飞书机器人 Webhook
// webhook: 飞书机器人 Webhook URL
// title: 消息标题（会与 content 拼接为完整文本）
// content: 消息正文
func (s *FeishuBotService) SendText(webhook string, title string, content string) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return fmt.Errorf("飞书 Webhook 不能为空")
	}

	text := title
	if content != "" {
		text = title + "\n\n" + content
	}

	body := feishuBotRequest{
		MsgType: "text",
		Content: feishuBotTextContent{Text: text},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化飞书消息失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= feishuBotRetries; attempt++ {
		lastErr = s.doPost(webhook, payload)
		if lastErr == nil {
			return nil
		}
		if attempt < feishuBotRetries {
			logger.Warn("飞书机器人发送失败，重试中: %v", lastErr)
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	return lastErr
}

func (s *FeishuBotService) doPost(webhook string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求飞书 Webhook 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("飞书 Webhook 返回非 200: status=%d, body=%s", resp.StatusCode, string(data))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取飞书响应失败: %w", err)
	}

	var r feishuBotResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("解析飞书响应失败: %w", err)
	}
	if r.Code != 0 {
		return fmt.Errorf("飞书返回错误: code=%d, msg=%s", r.Code, r.Msg)
	}
	return nil
}

// SendViaAPI 通过飞书开放平台「发送消息」API 发送文本
// 参考 https://open.feishu.cn/document/server-docs/im-v1/message/create
func (s *FeishuBotService) SendViaAPI(appID, appSecret, receiveIDType, receiveID, title, content string) error {
	appID = strings.TrimSpace(appID)
	appSecret = strings.TrimSpace(appSecret)
	receiveIDType = strings.TrimSpace(receiveIDType)
	receiveID = strings.TrimSpace(receiveID)
	if appID == "" || appSecret == "" || receiveID == "" {
		return fmt.Errorf("飞书 API 配置不完整：app_id、app_secret、receive_id 不能为空")
	}
	validTypes := map[string]bool{"chat_id": true, "user_id": true, "open_id": true}
	if !validTypes[receiveIDType] {
		receiveIDType = "chat_id"
	}

	text := title
	if content != "" {
		text = title + "\n\n" + content
	}

	token, err := s.getTenantAccessToken(appID, appSecret)
	if err != nil {
		return err
	}

	contentJSON, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("序列化消息内容失败: %w", err)
	}

	body := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveIDType
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	var lastErr error
	for attempt := 0; attempt <= feishuBotRetries; attempt++ {
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求飞书 API 失败: %w", err)
			if attempt < feishuBotRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("飞书 API 返回非 200: status=%d, body=%s", resp.StatusCode, string(data))
			if attempt < feishuBotRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		var apiResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.Unmarshal(data, &apiResp); err != nil {
			lastErr = fmt.Errorf("解析飞书 API 响应失败: %w", err)
			break
		}
		if apiResp.Code != 0 {
			lastErr = fmt.Errorf("飞书 API 错误: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
			if apiResp.Code == 99991663 || apiResp.Code == 99991664 {
				s.invalidateToken(appID)
			}
			if attempt < feishuBotRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		return nil
	}
	return lastErr
}

// SendToUserByOpenID 使用配置的 app_id/app_secret 向指定 open_id 用户发送消息（接收者为 users.feishu_id）
func (s *FeishuBotService) SendToUserByOpenID(openID string, title string, content string) error {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return fmt.Errorf("飞书 open_id 不能为空")
	}
	if s.feishuCfg == nil || strings.TrimSpace(s.feishuCfg.AppID) == "" || strings.TrimSpace(s.feishuCfg.AppSecret) == "" {
		return fmt.Errorf("飞书应用配置未设置（app_id、app_secret）")
	}
	return s.SendViaAPI(s.feishuCfg.AppID, s.feishuCfg.AppSecret, "open_id", openID, title, content)
}

func (s *FeishuBotService) getTenantAccessToken(appID, appSecret string) (string, error) {
	s.tokenMu.RLock()
	e, ok := s.tokenCache[appID]
	if ok && e.expires.After(time.Now().Add(feishuTokenRefreshBef)) {
		token := e.token
		s.tokenMu.RUnlock()
		return token, nil
	}
	s.tokenMu.RUnlock()

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	e, ok = s.tokenCache[appID]
	if ok && e.expires.After(time.Now().Add(feishuTokenRefreshBef)) {
		return e.token, nil
	}

	reqBody := map[string]string{"app_id": appID, "app_secret": appSecret}
	payload, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("创建 token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 tenant_access_token 失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 token 响应失败: %w", err)
	}
	var r struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("解析 token 响应失败: %w", err)
	}
	if r.Code != 0 {
		return "", fmt.Errorf("飞书 token 接口错误: code=%d, msg=%s", r.Code, r.Msg)
	}
	expireSec := r.Expire
	if expireSec <= 0 {
		expireSec = 7200
	}
	s.tokenCache[appID] = &tokenEntry{token: r.TenantAccessToken, expires: time.Now().Add(time.Duration(expireSec) * time.Second)}
	return r.TenantAccessToken, nil
}

func (s *FeishuBotService) invalidateToken(appID string) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	delete(s.tokenCache, appID)
}
