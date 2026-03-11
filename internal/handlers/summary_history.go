package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type SummaryHistoryHandler struct {
	svc *services.SummaryHistoryService
}

func NewSummaryHistoryHandler(svc *services.SummaryHistoryService) *SummaryHistoryHandler {
	return &SummaryHistoryHandler{svc: svc}
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
	AIModelID    uint   `json:"ai_model_id" binding:"required"`
	FeedIDs      []uint `json:"feed_ids"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Page         int    `json:"page"`
	PageSize     int    `json:"page_size"`
	Order        string `json:"order"`
	ArticleCount int    `json:"article_count"`
	Total        int64  `json:"total"`
	Content      string `json:"content"`
	Error        string `json:"error"`
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
		AIModelID:    req.AIModelID,
		FeedIDs:      req.FeedIDs,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Page:         req.Page,
		PageSize:     req.PageSize,
		Order:        req.Order,
		ArticleCount: req.ArticleCount,
		Total:        req.Total,
		Content:      req.Content,
		Error:        req.Error,
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

