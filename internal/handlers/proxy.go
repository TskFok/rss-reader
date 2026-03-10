package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type ProxyHandler struct {
	svc *services.ProxyService
}

func NewProxyHandler(svc *services.ProxyService) *ProxyHandler {
	return &ProxyHandler{svc: svc}
}

// List 代理列表
// GET /api/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取代理列表失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// Create 创建代理
// POST /api/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateProxyRequest
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

// Update 更新代理
// PUT /api/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Update(userID, uint(id), req)
	if err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete 删除代理
// DELETE /api/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除代理失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
