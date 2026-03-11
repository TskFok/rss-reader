package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/services"
)

type CategoryHandler struct {
	svc *services.CategoryService
}

func NewCategoryHandler(svc *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// List 分类列表
// GET /api/categories
func (h *CategoryHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// Create 创建分类
// POST /api/categories
func (h *CategoryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req services.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Create(userID, req)
	if err != nil {
		if err == services.ErrCategoryNameExists {
			c.JSON(http.StatusConflict, gin.H{"error": "分类名称已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分类失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// Update 更新分类
// PUT /api/categories/:id
func (h *CategoryHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	var req services.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	item, err := h.svc.Update(userID, uint(id), req)
	if err != nil {
		switch err {
		case services.ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
		case services.ErrCategoryNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": "分类名称已存在"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分类失败"})
		}
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete 删除分类
// DELETE /api/categories/:id
func (h *CategoryHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.svc.Delete(userID, uint(id)); err != nil {
		if err == services.ErrCategoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分类失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// CategoryReorderRequest 分类排序请求
type CategoryReorderRequest struct {
	IDList []uint `json:"id_list" binding:"required"`
}

// Reorder 拖动排序
// PUT /api/categories/reorder
func (h *CategoryHandler) Reorder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req CategoryReorderRequest
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

