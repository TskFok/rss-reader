package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/services"
)

const (
	UserIDKey   = "user_id"
	UserKey     = "user"
)

// Auth 从 Authorization: Bearer <token> 提取并验证 JWT
func Auth(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证信息"})
			c.Abort()
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}
		userID, err := authSvc.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证无效或已过期"})
			c.Abort()
			return
		}
		c.Set(UserIDKey, userID)
		c.Next()
	}
}

// GetUserID 从上下文获取当前用户 ID
func GetUserID(c *gin.Context) uint {
	v, _ := c.Get(UserIDKey)
	if v == nil {
		return 0
	}
	return v.(uint)
}
