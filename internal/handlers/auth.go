package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tskfok/rss-reader/internal/services"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authSvc *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register 注册
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	user, err := h.authSvc.Register(req)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功，请等待管理员解锁",
		"user":    user,
	})
}

// Login 登录
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	// 使用通用绑定，兼容 application/json、application/x-www-form-urlencoded、
	// multipart/form-data，避免仅按 JSON 解析导致的参数错误。
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	res, err := h.authSvc.Login(req)
	if err != nil {
		switch err {
		case services.ErrUserLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "用户已被锁定，请联系管理员解锁"})
		case services.ErrInvalidCreds:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		}
		return
	}
	c.JSON(http.StatusOK, res)
}
