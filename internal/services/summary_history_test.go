package services

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupSummaryHistoryDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AIModel{}, &models.AISummaryHistory{}))
	return db
}

func TestSummaryHistoryService_CreateListDelete(t *testing.T) {
	db := setupSummaryHistoryDB(t)
	hsvc := NewSummaryHistoryService(db)

	// seed user + model
	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "m", BaseURL: "http://x"}
	require.NoError(t, db.Create(&m).Error)

	now := time.Now()
	_, err := hsvc.Create(u.ID, CreateSummaryHistoryRequest{
		AIModelID:    m.ID,
		FeedIDs:      []uint{1, 2},
		StartTime:    "2026-03-01",
		EndTime:      "2026-03-02",
		Page:         1,
		PageSize:     20,
		Order:        "desc",
		ArticleCount: 2,
		Total:        10,
		Content:      "hello",
		Error:        "",
		CreatedAt:    &now,
	})
	require.NoError(t, err)

	items, total, err := hsvc.List(u.ID, ListSummaryHistoriesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "m", items[0].AIModelName)
	assert.Equal(t, "hello", items[0].Content)
	assert.Equal(t, 1, items[0].Page)
	assert.Equal(t, 20, items[0].PageSize)

	// delete ok
	require.NoError(t, hsvc.Delete(u.ID, items[0].ID))

	_, total2, err := hsvc.List(u.ID, ListSummaryHistoriesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total2)
}

func TestSummaryHistoryService_Delete_OtherUser(t *testing.T) {
	db := setupSummaryHistoryDB(t)
	hsvc := NewSummaryHistoryService(db)

	u1 := models.User{Username: "u1", PasswordHash: "h", Status: models.UserStatusActive}
	u2 := models.User{Username: "u2", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)
	m := models.AIModel{UserID: u1.ID, Name: "m", BaseURL: "http://x"}
	require.NoError(t, db.Create(&m).Error)

	h, err := hsvc.Create(u1.ID, CreateSummaryHistoryRequest{AIModelID: m.ID, Content: "x"})
	require.NoError(t, err)

	err = hsvc.Delete(u2.ID, h.ID)
	assert.ErrorIs(t, err, ErrSummaryHistoryNotFound)
}

