package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type SummaryHistoryHandler struct {
	svc         *services.SummaryHistoryService
	articleSvc  *services.ArticleService
	aiModelSvc  *services.AIModelService
	templateSvc *services.SummaryTemplateService
}

func NewSummaryHistoryHandler(svc *services.SummaryHistoryService, articleSvc *services.ArticleService, aiModelSvc *services.AIModelService, templateSvc *services.SummaryTemplateService) *SummaryHistoryHandler {
	return &SummaryHistoryHandler{svc: svc, articleSvc: articleSvc, aiModelSvc: aiModelSvc, templateSvc: templateSvc}
}

// List GET /api/summary-histories
func (h *SummaryHistoryHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.ListSummaryHistoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	items, total, err := h.svc.List(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取总结历史失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

type CreateSummaryHistoryRequest struct {
	AIModelID           uint   `json:"ai_model_id" binding:"required"`
	SummaryTemplateID   *uint  `json:"summary_template_id"`
	SummaryTemplateName string `json:"summary_template_name"`
	FeedIDs             []uint `json:"feed_ids"`
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	Page                int    `json:"page"`
	PageSize            int    `json:"page_size"`
	Order               string `json:"order"`
	ArticleCount        int    `json:"article_count"`
	Total               int64  `json:"total"`
	Content             string `json:"content"`
	Error               string `json:"error"`
}

// Create POST /api/summary-histories
func (h *SummaryHistoryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req CreateSummaryHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Create(userID, services.CreateSummaryHistoryRequest{
		AIModelID:           req.AIModelID,
		SummaryTemplateID:   req.SummaryTemplateID,
		SummaryTemplateName: req.SummaryTemplateName,
		FeedIDs:             req.FeedIDs,
		StartTime:           req.StartTime,
		EndTime:             req.EndTime,
		Page:                req.Page,
		PageSize:            req.PageSize,
		Order:               req.Order,
		ArticleCount:        req.ArticleCount,
		Total:               req.Total,
		Content:             req.Content,
		Error:               req.Error,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": item.ID})
}

// Delete DELETE /api/summary-histories/:id
func (h *SummaryHistoryHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrSummaryHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Retry POST /api/summary-histories/:id/retry
// 根据失败历史记录的查询条件，重新调用 AI 总结并覆盖更新原历史记录。
func (h *SummaryHistoryHandler) Retry(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	his, err := h.svc.GetByID(userID, uint(id))
	if err != nil {
		if err == services.ErrSummaryHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取记录失败"})
		return
	}

	var feedIDs []uint
	if his.FeedIDsJSON != "" {
		_ = json.Unmarshal([]byte(his.FeedIDsJSON), &feedIDs)
	}

	var startTime, endTime *time.Time
	if his.StartTime != "" {
		t, parseErr := time.Parse("2006-01-02", his.StartTime)
		if parseErr == nil {
			startTime = &t
		}
	}
	if his.EndTime != "" {
		t, parseErr := time.Parse("2006-01-02", his.EndTime)
		if parseErr == nil {
			t = t.Add(24*time.Hour - time.Second)
			endTime = &t
		}
	}

	articles, total, err := h.articleSvc.ListForSummary(userID, feedIDs, startTime, endTime, his.Page, his.PageSize, his.Order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		return
	}
	if len(articles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "指定条件下没有文章可总结"})
		return
	}

	var prompt *services.SummaryPromptOptions
	if h.templateSvc != nil && his.SummaryTemplateID != nil && *his.SummaryTemplateID != 0 {
		if t, err := h.templateSvc.GetByID(userID, *his.SummaryTemplateID); err == nil {
			prompt = services.PromptOptionsFromTemplate(t)
		}
	}
	content, sumErr := h.aiModelSvc.Summarize(userID, his.AIModelID, articles, prompt)
	errStr := ""
	if sumErr != nil {
		errStr = sumErr.Error()
	}

	updated, updateErr := h.svc.UpdateResult(userID, his.ID, services.UpdateSummaryHistoryResultRequest{
		ArticleCount: len(articles),
		Total:        total,
		Content:      content,
		Error:        errStr,
	})
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      updated.ID,
		"content": updated.Content,
		"error":   updated.Error,
	})
}

