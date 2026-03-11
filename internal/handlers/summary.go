package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

// SummaryHandler AI 总结处理器
type SummaryHandler struct {
	articleSvc *services.ArticleService
	aiModelSvc *services.AIModelService
}

// NewSummaryHandler 创建总结处理器
func NewSummaryHandler(articleSvc *services.ArticleService, aiModelSvc *services.AIModelService) *SummaryHandler {
	return &SummaryHandler{articleSvc: articleSvc, aiModelSvc: aiModelSvc}
}

// SummarizeRequest 总结请求
type SummarizeRequest struct {
	AIModelID uint     `json:"ai_model_id" binding:"required"`
	FeedIDs   []uint   `json:"feed_ids"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

// Summarize 流式生成 AI 总结（SSE）
// POST /api/articles/summarize
func (h *SummaryHandler) Summarize(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	var startTime, endTime *time.Time
	if req.StartTime != "" {
		t, err := time.Parse("2006-01-02", req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time 格式错误，请使用 YYYY-MM-DD"})
			return
		}
		startTime = &t
	}
	if req.EndTime != "" {
		t, err := time.Parse("2006-01-02", req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_time 格式错误，请使用 YYYY-MM-DD"})
			return
		}
		// 包含结束日期的全天
		t = t.Add(24*time.Hour - time.Second)
		endTime = &t
	}
	articles, err := h.articleSvc.ListForSummary(userID, req.FeedIDs, startTime, endTime, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		return
	}
	if len(articles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "指定条件下没有文章可总结"})
		return
	}
	// 先检查模型是否存在
	if _, err := h.aiModelSvc.GetByID(userID, req.AIModelID); err != nil {
		if err == services.ErrAIModelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI 模型不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型失败"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()
	// 先发送 article_count（传 map 避免 Gin 对字符串二次 JSON 编码）
	c.SSEvent("", map[string]interface{}{"article_count": len(articles)})
	c.Writer.Flush()
	// 流式输出
	err = h.aiModelSvc.SummarizeStream(userID, req.AIModelID, articles, func(chunk string) error {
		c.SSEvent("", map[string]string{"delta": chunk})
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		c.SSEvent("", map[string]string{"error": err.Error()})
		c.Writer.Flush()
	}
}
