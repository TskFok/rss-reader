package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

// UserSettingHandler 用户设置处理器
type UserSettingHandler struct {
	userSettingSvc *services.UserSettingService
	feishuBot      services.FeishuBotClient
}

// NewUserSettingHandler 创建用户设置处理器
func NewUserSettingHandler(userSettingSvc *services.UserSettingService, feishuBot services.FeishuBotClient) *UserSettingHandler {
	return &UserSettingHandler{
		userSettingSvc: userSettingSvc,
		feishuBot:      feishuBot,
	}
}

// GetSettingsResponse 获取设置响应
type GetSettingsResponse struct {
	FeishuNotifyType string `json:"feishu_notify_type"`
	FeishuBotWebhook string `json:"feishu_bot_webhook"`
	FeishuID         string `json:"feishu_id"` // 用户绑定的飞书 open_id，API 模式接收者
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	FeishuNotifyType *string `json:"feishu_notify_type"`
	FeishuBotWebhook *string `json:"feishu_bot_webhook"`
}

// GetSettings 获取当前用户设置
// GET /api/users/me/settings
func (h *UserSettingHandler) GetSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cfg, err := h.userSettingSvc.GetFeishuNotifyConfig(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设置失败"})
		return
	}
	c.JSON(http.StatusOK, GetSettingsResponse{
		FeishuNotifyType: cfg.NotifyType,
		FeishuBotWebhook: cfg.Webhook,
		FeishuID:         cfg.FeishuID,
	})
}

// UpdateSettings 更新当前用户设置
// PUT /api/users/me/settings
func (h *UserSettingHandler) UpdateSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	cfg, err := h.userSettingSvc.GetFeishuNotifyConfig(userID)
	if err != nil {
		cfg = &services.FeishuNotifyConfig{}
	}
	if req.FeishuNotifyType != nil {
		cfg.NotifyType = *req.FeishuNotifyType
	}
	if req.FeishuBotWebhook != nil {
		cfg.Webhook = *req.FeishuBotWebhook
		// 仅当未显式指定类型时，才根据 webhook 推断为 webhook（避免覆盖用户选择的 api）
		if *req.FeishuBotWebhook != "" && (req.FeishuNotifyType == nil || *req.FeishuNotifyType == "") {
			cfg.NotifyType = "webhook"
		}
	}
	if err := h.userSettingSvc.UpdateFeishuNotifyConfig(userID, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设置失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

// TestFeishuBot 测试发送飞书机器人消息
// POST /api/users/me/feishu-bot/test
func (h *UserSettingHandler) TestFeishuBot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cfg, err := h.userSettingSvc.GetFeishuNotifyConfigForSend(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设置失败"})
		return
	}
	notifyType := cfg.NotifyType
	if notifyType == "" && cfg.Webhook != "" {
		notifyType = "webhook"
	}
	if notifyType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置飞书通知方式（Webhook 或服务端 API）"})
		return
	}
	if h.feishuBot == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "飞书机器人服务不可用"})
		return
	}
	title := "RSS Reader 测试消息"
	content := "这是一条来自 RSS Reader 的测试消息，表示飞书机器人配置正确。"
	var sendErr error
	if notifyType == "webhook" && cfg.Webhook != "" {
		sendErr = h.feishuBot.SendText(cfg.Webhook, title, content)
	} else if notifyType == "api" && cfg.FeishuID != "" {
		sendErr = h.feishuBot.SendToUserByOpenID(cfg.FeishuID, title, content)
	} else if notifyType == "api" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务端 API 模式需先绑定飞书账号，告警将发送到您的飞书私聊"})
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置飞书通知方式（Webhook 或服务端 API）"})
		return
	}
	if sendErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发送失败: " + sendErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试消息已发送"})
}
