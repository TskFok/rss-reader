package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type ErrorLogHandler struct {
	svc *services.ErrorLogService
}

func NewErrorLogHandler(svc *services.ErrorLogService) *ErrorLogHandler {
	return &ErrorLogHandler{svc: svc}
}

// List GET /api/error-logs
func (h *ErrorLogHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.ListErrorLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	items, total, err := h.svc.List(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取错误日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// Delete DELETE /api/error-logs/:id
func (h *ErrorLogHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

