package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type AIModelHandler struct {
	svc *services.AIModelService
}

func NewAIModelHandler(svc *services.AIModelService) *AIModelHandler {
	return &AIModelHandler{svc: svc}
}

// List AI 模型列表
// GET /api/ai-models
func (h *AIModelHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 AI 模型列表失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// Create 创建 AI 模型
// POST /api/ai-models
func (h *AIModelHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Create(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// Update 更新 AI 模型
// PUT /api/ai-models/:id
func (h *AIModelHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Update(userID, uint(id), req)
	if err != nil {
		if err == services.ErrAIModelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI 模型不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete 删除 AI 模型
// DELETE /api/ai-models/:id
func (h *AIModelHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrAIModelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI 模型不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除 AI 模型失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ReorderRequest 排序请求
type ReorderRequest struct {
	IDList []uint `json:"id_list" binding:"required"`
}

// Reorder 拖动排序
// PUT /api/ai-models/reorder
func (h *AIModelHandler) Reorder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.svc.Reorder(userID, req.IDList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排序失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "排序已更新"})
}

// Test 检测 AI 模型是否可用
// POST /api/ai-models/:id/test
func (h *AIModelHandler) Test(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Test(userID, uint(id)); err != nil {
		if err == services.ErrAIModelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI 模型不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "模型可用"})
}
