package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/services"
	"gorm.io/gorm"
)

func setupSummaryHistoryRetryHandlers(t *testing.T, aiBaseURL string) (*gin.Engine, *gorm.DB, uint, uint) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Feed{},
		&models.Article{},
		&models.UserArticle{},
		&models.AIModel{},
		&models.AISummaryHistory{},
	))

	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)

	m := models.AIModel{UserID: u.ID, Name: "m", BaseURL: aiBaseURL}
	require.NoError(t, db.Create(&m).Error)

	articleSvc := services.NewArticleService(db)
	historySvc := services.NewSummaryHistoryService(db)
	aiModelSvc := services.NewAIModelService(db)
	h := NewSummaryHistoryHandler(historySvc, articleSvc, aiModelSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, u.ID)
		c.Next()
	})
	r.POST("/api/summary-histories/:id/retry", h.Retry)

	return r, db, u.ID, m.ID
}

func TestSummaryHistoryHandler_Retry_CreatesNewHistory(t *testing.T) {
	// mock AI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"重试总结成功"}}]}`))
	}))
	defer server.Close()

	r, db, userID, modelID := setupSummaryHistoryRetryHandlers(t, server.URL)

	feed := models.Feed{UserID: userID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	published := time.Date(2026, 3, 10, 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &published}
	require.NoError(t, db.Create(&a1).Error)

	// seed a failed history (same query condition)
	hsvc := services.NewSummaryHistoryService(db)
	old, err := hsvc.Create(userID, services.CreateSummaryHistoryRequest{
		AIModelID:    modelID,
		FeedIDs:      []uint{feed.ID},
		StartTime:    "2026-03-10",
		EndTime:      "2026-03-10",
		Page:         1,
		PageSize:     20,
		Order:        "desc",
		ArticleCount: 0,
		Total:        0,
		Content:      "",
		Error:        "boom",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/summary-histories/"+strconv.FormatUint(uint64(old.ID), 10)+"/retry", nil))

	assert.Equal(t, http.StatusCreated, w.Code)

	var count int64
	require.NoError(t, db.Model(&models.AISummaryHistory{}).Where("user_id = ?", userID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

