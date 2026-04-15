package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/services"
	"gorm.io/gorm"
)

func setupArticleHandler(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))

	h := NewArticleHandler(services.NewArticleService(db), nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, uint(1))
		c.Next()
	})
	r.PUT("/api/articles/:id/read", h.MarkRead)
	return r, db
}

func TestArticleHandler_MarkRead(t *testing.T) {
	r, db := setupArticleHandler(t)

	user := models.User{ID: 1, Username: "alice", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	article := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g-handler"), GUIDRaw: "g-handler", Title: "a"}
	require.NoError(t, db.Create(&article).Error)

	req := httptest.NewRequest(http.MethodPut, "/api/articles/"+jsonUint(article.ID)+"/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "已标记为已读", resp["message"])

	var ua models.UserArticle
	require.NoError(t, db.Where("user_id = ? AND article_id = ?", user.ID, article.ID).First(&ua).Error)
	assert.True(t, ua.ReadStatus)
}

func TestArticleHandler_MarkRead_NotFound(t *testing.T) {
	r, _ := setupArticleHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/api/articles/999/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "文章不存在", resp["error"])
}

func jsonUint(v uint) string {
	return fmt.Sprintf("%d", v)
}
