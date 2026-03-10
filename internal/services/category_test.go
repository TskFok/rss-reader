package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupCategoryDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
	return db
}

func TestCategoryService_CRUD(t *testing.T) {
	db := setupCategoryDB(t)
	svc := NewCategoryService(db)

	// create
	c1, err := svc.Create(1, CreateCategoryRequest{Name: "科技"})
	require.NoError(t, err)
	assert.Equal(t, "科技", c1.Name)

	// list
	items, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, c1.ID, items[0].ID)

	// update
	updated, err := svc.Update(1, c1.ID, UpdateCategoryRequest{Name: "技术"})
	require.NoError(t, err)
	assert.Equal(t, "技术", updated.Name)

	// duplicate name per user
	_, err = svc.Create(1, CreateCategoryRequest{Name: "技术"})
	assert.ErrorIs(t, err, ErrCategoryNameExists)

	// same name allowed for different user
	_, err = svc.Create(2, CreateCategoryRequest{Name: "技术"})
	require.NoError(t, err)

	// delete
	err = svc.Delete(1, c1.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(1, c1.ID)
	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

