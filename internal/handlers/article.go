package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	articleSvc *services.ArticleService
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(articleSvc *services.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleSvc: articleSvc}
}

// List 文章列表
// GET /api/articles
func (h *ArticleHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.ListArticlesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if c.Query("feed_id") != "" {
		id, err := strconv.ParseUint(c.Query("feed_id"), 10, 32)
		if err == nil {
			uid := uint(id)
			req.FeedID = &uid
		}
	}
	if c.Query("read") == "true" {
		b := true
		req.Read = &b
	} else if c.Query("read") == "false" {
		b := false
		req.Read = &b
	}
	if c.Query("favorite") == "true" {
		b := true
		req.Favorite = &b
	}
	articles, total, err := h.articleSvc.List(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": articles,
		"total": total,
	})
}

// MarkRead 标记已读
// PUT /api/articles/:id/read
func (h *ArticleHandler) MarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.articleSvc.MarkRead(userID, uint(id)); err != nil {
		if err == services.ErrArticleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已标记为已读"})
}

// ToggleFavorite 切换收藏
// PUT /api/articles/:id/favorite
func (h *ArticleHandler) ToggleFavorite(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	favorite, err := h.articleSvc.ToggleFavorite(userID, uint(id))
	if err != nil {
		if err == services.ErrArticleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorite": favorite})
}
