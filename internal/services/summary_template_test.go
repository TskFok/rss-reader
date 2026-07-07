package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupSummaryTemplateDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AISummaryTemplate{}))
	return db
}

func TestSummaryTemplateService_CRUD(t *testing.T) {
	db := setupSummaryTemplateDB(t)
	svc := NewSummaryTemplateService(db)
	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)

	created, err := svc.Create(u.ID, CreateSummaryTemplateRequest{
		Name:             "技术简报",
		SystemPrompt:     "你是助手",
		UserPromptPrefix: "请输出要点列表",
	})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)

	items, err := svc.List(u.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)

	got, err := svc.GetByID(u.ID, created.ID)
	require.NoError(t, err)
	opts := PromptOptionsFromTemplate(got)
	require.NotNil(t, opts)
	assert.Equal(t, "你是助手", opts.SystemPrompt)
	assert.Equal(t, "请输出要点列表", opts.UserPrefix)

	_, err = svc.Update(u.ID, created.ID, UpdateSummaryTemplateRequest{
		Name:             "技术简报2",
		SystemPrompt:     "",
		UserPromptPrefix: "短评",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(u.ID, created.ID))
	_, err = svc.GetByID(u.ID, created.ID)
	assert.ErrorIs(t, err, ErrSummaryTemplateNotFound)
}
