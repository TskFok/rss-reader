package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupSummaryRunnerDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.FeedCategory{},
		&models.Feed{},
		&models.Article{},
		&models.UserArticle{},
		&models.Proxy{},
		&models.AIModel{},
		&models.AISummaryHistory{},
	))
	return db
}

func TestRunDailySummaryForYesterday_CreatesHistoriesUntilEmpty(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	// mock AI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"这是中文总结内容。"}}]}`))
	}))
	defer server.Close()

	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "test-model", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	// create two articles yesterday (Shanghai date)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	t2 := time.Date(y.Year(), y.Month(), y.Day(), 12, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "a2", Content: "c2", PublishedAt: &t2}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	// page_size=1 -> should generate 2 histories then stop at empty page
	err = RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.AISummaryHistory{}).Where("user_id = ?", u.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

