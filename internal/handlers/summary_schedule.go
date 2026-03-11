package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type SummaryScheduleHandler struct {
	svc *services.SummaryScheduleService
}

func NewSummaryScheduleHandler(svc *services.SummaryScheduleService) *SummaryScheduleHandler {
	return &SummaryScheduleHandler{svc: svc}
}

func (h *SummaryScheduleHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取定时配置失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *SummaryScheduleHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateSummaryScheduleRequest
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

func (h *SummaryScheduleHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateSummaryScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Update(userID, uint(id), req)
	if err != nil {
		if err == services.ErrSummaryScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *SummaryScheduleHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrSummaryScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

