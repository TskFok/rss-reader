package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/config"
)

func TestFeishuBotService_SendText_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	err := svc.SendText(server.URL, "测试标题", "测试内容")
	require.NoError(t, err)

	var body feishuBotRequest
	require.NoError(t, json.Unmarshal(receivedBody, &body))
	assert.Equal(t, "text", body.MsgType)
	assert.Equal(t, "测试标题\n\n测试内容", body.Content.Text)
}

func TestFeishuBotService_SendText_OnlyTitle(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	err := svc.SendText(server.URL, "仅标题", "")
	require.NoError(t, err)

	var body feishuBotRequest
	require.NoError(t, json.Unmarshal(receivedBody, &body))
	assert.Equal(t, "仅标题", body.Content.Text)
}

func TestFeishuBotService_SendText_FeishuError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":9499,"msg":"invalid webhook url"}`))
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	err := svc.SendText(server.URL, "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "飞书返回错误")
	assert.Contains(t, err.Error(), "9499")
	assert.Contains(t, err.Error(), "invalid webhook url")
}

func TestFeishuBotService_SendText_EmptyWebhook(t *testing.T) {
	svc := NewFeishuBotService(nil)
	err := svc.SendText("", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Webhook 不能为空")
}

func TestFeishuBotService_SendText_HTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	err := svc.SendText(server.URL, "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非 200")
}

func TestFeishuBotService_SendText_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	err := svc.SendText(server.URL, "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析飞书响应")
}

func TestFeishuBotService_SendText_WebhookWhitespaceTrimmed(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	svc := NewFeishuBotService(nil)
	// Webhook 前后有空白，应被 trim 后正常发送
	err := svc.SendText("  "+server.URL+"  ", "标题", "内容")
	require.NoError(t, err)

	var body feishuBotRequest
	require.NoError(t, json.Unmarshal(receivedBody, &body))
	assert.Equal(t, "标题\n\n内容", body.Content.Text)
}

func TestFeishuBotService_SendText_WhitespaceOnlyWebhook(t *testing.T) {
	svc := NewFeishuBotService(nil)
	err := svc.SendText("   \t  ", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Webhook 不能为空")
}

func TestFeishuBotService_SendViaAPI_EmptyConfig(t *testing.T) {
	svc := NewFeishuBotService(&config.FeishuConfig{})
	err := svc.SendViaAPI("", "secret", "chat_id", "oc_xxx", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "配置不完整")

	err = svc.SendViaAPI("cli_x", "", "chat_id", "oc_xxx", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "配置不完整")

	err = svc.SendViaAPI("cli_x", "secret", "chat_id", "", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "配置不完整")
}

func TestFeishuBotService_SendToUserByOpenID_EmptyConfig(t *testing.T) {
	svc := NewFeishuBotService(nil)
	err := svc.SendToUserByOpenID("open_xxx", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "飞书应用配置未设置")
}

func TestFeishuBotService_SendToUserByOpenID_EmptyOpenID(t *testing.T) {
	svc := NewFeishuBotService(&config.FeishuConfig{AppID: "cli", AppSecret: "secret"})
	err := svc.SendToUserByOpenID("", "标题", "内容")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open_id 不能为空")
}
