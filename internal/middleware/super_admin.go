package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

// RequireSuperAdmin 要求当前用户为超级管理员
func RequireSuperAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(UserIDKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}
		if !user.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要超级管理员权限"})
			c.Abort()
			return
		}
		c.Set(UserKey, &user)
		c.Next()
	}
}
