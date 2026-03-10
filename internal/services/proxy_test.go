package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupProxyDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Proxy{}))
	return db
}

func TestProxyService_CRUD(t *testing.T) {
	db := setupProxyDB(t)
	svc := NewProxyService(db)

	// create
	p1, err := svc.Create(1, CreateProxyRequest{
		Name: "本地代理",
		URL:  "http://127.0.0.1:7890",
	})
	require.NoError(t, err)
	assert.Equal(t, "本地代理", p1.Name)
	assert.Equal(t, "http://127.0.0.1:7890", p1.URL)

	// list
	items, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, p1.ID, items[0].ID)

	// update
	updated, err := svc.Update(1, p1.ID, UpdateProxyRequest{
		Name: "Clash 代理",
		URL:  "http://127.0.0.1:7891",
	})
	require.NoError(t, err)
	assert.Equal(t, "Clash 代理", updated.Name)
	assert.Equal(t, "http://127.0.0.1:7891", updated.URL)

	// create another for different user
	p2, err := svc.Create(2, CreateProxyRequest{
		URL: "socks5://proxy.example.com:1080",
	})
	require.NoError(t, err)
	assert.Equal(t, "", p2.Name)
	assert.Equal(t, "socks5://proxy.example.com:1080", p2.URL)

	// list only returns own user's proxies
	items1, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, items1, 1)

	items2, err := svc.List(2)
	require.NoError(t, err)
	require.Len(t, items2, 1)

	// delete
	err = svc.Delete(1, p1.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(1, p1.ID)
	assert.ErrorIs(t, err, ErrProxyNotFound)

	// delete other user's proxy should fail (not found for user 1)
	err = svc.Delete(1, p2.ID)
	assert.ErrorIs(t, err, ErrProxyNotFound)
}

func TestProxyService_EmptyURL(t *testing.T) {
	db := setupProxyDB(t)
	svc := NewProxyService(db)

	_, err := svc.Create(1, CreateProxyRequest{URL: "   "})
	assert.Error(t, err)

	p, err := svc.Create(1, CreateProxyRequest{URL: "http://a:1"})
	require.NoError(t, err)

	_, err = svc.Update(1, p.ID, UpdateProxyRequest{URL: ""})
	assert.Error(t, err)
}
