package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupSummaryScheduleDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AIModel{}, &models.AISummarySchedule{}))
	return db
}

func TestSummaryScheduleService_CRUD(t *testing.T) {
	db := setupSummaryScheduleDB(t)
	svc := NewSummaryScheduleService(db)

	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "m", BaseURL: "http://x"}
	require.NoError(t, db.Create(&m).Error)

	created, err := svc.Create(u.ID, CreateSummaryScheduleRequest{
		AIModelID: m.ID,
		FeedIDs:   []uint{1, 2},
		RunAt:     "08:30",
		PageSize:  50,
		Order:     "desc",
	})
	require.NoError(t, err)
	assert.Equal(t, "08:30", created.RunAt)
	assert.Equal(t, 50, created.PageSize)

	items, err := svc.List(u.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)

	disabled := false
	updated, err := svc.Update(u.ID, created.ID, UpdateSummaryScheduleRequest{
		AIModelID: m.ID,
		FeedIDs:   []uint{},
		RunAt:     "09:00",
		PageSize:  20,
		Order:     "asc",
		Enabled:   &disabled,
	})
	require.NoError(t, err)
	assert.Equal(t, "09:00", updated.RunAt)
	assert.Equal(t, "asc", updated.Order)
	assert.False(t, updated.Enabled)

	require.NoError(t, svc.Delete(u.ID, created.ID))
	after, err := svc.List(u.ID)
	require.NoError(t, err)
	assert.Len(t, after, 0)
}

func TestSummaryScheduleService_RunAtValidation(t *testing.T) {
	db := setupSummaryScheduleDB(t)
	svc := NewSummaryScheduleService(db)

	_, err := svc.Create(1, CreateSummaryScheduleRequest{AIModelID: 1, RunAt: "bad"})
	assert.Error(t, err)
}

