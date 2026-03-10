package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

// FeedHandler 订阅处理器
type FeedHandler struct {
	feedSvc *services.FeedService
}

// NewFeedHandler 创建订阅处理器
func NewFeedHandler(feedSvc *services.FeedService) *FeedHandler {
	return &FeedHandler{feedSvc: feedSvc}
}

// List 订阅列表
// GET /api/feeds
func (h *FeedHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	feeds, err := h.feedSvc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取订阅列表失败"})
		return
	}
	c.JSON(http.StatusOK, feeds)
}

// Create 添加订阅
// POST /api/feeds
func (h *FeedHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	feed, err := h.feedSvc.Create(userID, req)
	if err != nil {
		if err == services.ErrInvalidFeedURL {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 RSS 地址"})
			return
		}
		if err.Error() == "分类不存在" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分类不存在"})
			return
		}
		if err.Error() == "订阅已存在" {
			c.JSON(http.StatusConflict, gin.H{"error": "订阅已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加订阅失败"})
		return
	}
	c.JSON(http.StatusCreated, feed)
}

// Update 更新订阅设置
// PUT /api/feeds/:id
func (h *FeedHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	feed, err := h.feedSvc.Update(userID, uint(id), req)
	if err != nil {
		if err == services.ErrFeedNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "订阅不存在"})
			return
		}
		if err.Error() == "代理不存在" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "代理不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, feed)
}

// Delete 删除订阅
// DELETE /api/feeds/:id
func (h *FeedHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.feedSvc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrFeedNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "订阅不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
