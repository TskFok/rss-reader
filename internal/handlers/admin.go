package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/services"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	adminSvc *services.AdminService
}

// NewAdminHandler 创建管理员处理器
func NewAdminHandler(adminSvc *services.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

// ListUsers 用户列表
// GET /api/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.adminSvc.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// UnlockUser 解锁用户
// PUT /api/admin/users/:id/unlock
func (h *AdminHandler) UnlockUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.adminSvc.UnlockUser(uint(id)); err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解锁失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "解锁成功"})
}
