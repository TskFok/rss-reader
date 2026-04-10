package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type SummaryTemplateHandler struct {
	svc *services.SummaryTemplateService
}

func NewSummaryTemplateHandler(svc *services.SummaryTemplateService) *SummaryTemplateHandler {
	return &SummaryTemplateHandler{svc: svc}
}

func (h *SummaryTemplateHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模版列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *SummaryTemplateHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateSummaryTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	m, err := h.svc.Create(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *SummaryTemplateHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateSummaryTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	m, err := h.svc.Update(userID, uint(id), req)
	if err != nil {
		if err == services.ErrSummaryTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "模版不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *SummaryTemplateHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrSummaryTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "模版不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
