package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type manualAIBody struct {
	AIModelID        *uint  `json:"ai_model_id"`
	AITargetLanguage string `json:"ai_target_language"`
}

func bindManualAIBody(c *gin.Context, out *manualAIBody) error {
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

func (h *ArticleHandler) manualAIRespondError(c *gin.Context, userID uint, articleID uint, err error) {
	ar, errGet := h.articleSvc.GetWithRead(userID, articleID)
	payload := gin.H{"error": err.Error()}
	if errGet == nil && strings.TrimSpace(ar.AILastError) != "" {
		payload["ai_last_error"] = strings.TrimSpace(ar.AILastError)
	}
	status := http.StatusBadRequest
	if errors.Is(err, services.ErrArticleNotFound) {
		status = http.StatusNotFound
	}
	c.JSON(status, payload)
}

// ArticleHandler 文章处理器
type ArticleHandler struct {
	articleSvc *services.ArticleService
	articleAI  *services.ArticleAIProcessor
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(articleSvc *services.ArticleService, articleAI *services.ArticleAIProcessor) *ArticleHandler {
	return &ArticleHandler{articleSvc: articleSvc, articleAI: articleAI}
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

// ManualAIClassify POST /api/articles/:id/ai/classify
func (h *ArticleHandler) ManualAIClassify(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if h.articleAI == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务不可用"})
		return
	}
	var body manualAIBody
	if err := bindManualAIBody(c, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.articleAI.ManualClassify(userID, uint(id), body.AIModelID); err != nil {
		h.manualAIRespondError(c, userID, uint(id), err)
		return
	}
	ar, err := h.articleSvc.GetWithRead(userID, uint(id))
	if err != nil {
		if errors.Is(err, services.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"article": ar})
}

// ManualAITranslate POST /api/articles/:id/ai/translate
func (h *ArticleHandler) ManualAITranslate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if h.articleAI == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务不可用"})
		return
	}
	var body manualAIBody
	if err := bindManualAIBody(c, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.articleAI.ManualTranslate(userID, uint(id), body.AIModelID, body.AITargetLanguage); err != nil {
		h.manualAIRespondError(c, userID, uint(id), err)
		return
	}
	ar, err := h.articleSvc.GetWithRead(userID, uint(id))
	if err != nil {
		if errors.Is(err, services.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"article": ar})
}

// ManualAITranslateStream POST /api/articles/:id/ai/translate/stream
func (h *ArticleHandler) ManualAITranslateStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if h.articleAI == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务不可用"})
		return
	}
	var body manualAIBody
	if err := bindManualAIBody(c, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()

	err = h.articleAI.ManualTranslateStream(userID, uint(id), body.AIModelID, body.AITargetLanguage, func(delta string) error {
		c.SSEvent("", map[string]string{"delta": delta})
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		payload := gin.H{"error": err.Error()}
		if ar, errGet := h.articleSvc.GetWithRead(userID, uint(id)); errGet == nil && strings.TrimSpace(ar.AILastError) != "" {
			payload["ai_last_error"] = strings.TrimSpace(ar.AILastError)
		}
		c.SSEvent("", payload)
		c.Writer.Flush()
		return
	}

	ar, err := h.articleSvc.GetWithRead(userID, uint(id))
	if err != nil {
		if errors.Is(err, services.ErrArticleNotFound) {
			c.SSEvent("", map[string]string{"error": "文章不存在"})
			c.Writer.Flush()
			return
		}
		c.SSEvent("", map[string]string{"error": "获取文章失败"})
		c.Writer.Flush()
		return
	}
	c.SSEvent("", gin.H{"article": ar})
	c.Writer.Flush()
}
