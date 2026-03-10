package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupFeedDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Proxy{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
	return db
}

func TestFeedService_Create(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	// 分类不存在应优先失败（不会触发抓取）
	_, err := svc.Create(1, CreateFeedRequest{URL: "http://example.com/feed", CategoryID: 999, UpdateIntervalMinutes: 60})
	assert.Equal(t, "分类不存在", err.Error())

	// 构造本地 RSS 源，避免外网依赖
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>http://example.com/</link>
    <description>test</description>
    <item>
      <title>Hello</title>
      <link>http://example.com/hello</link>
      <guid>hello</guid>
    </item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	cat, err := NewCategoryService(db).Create(1, CreateCategoryRequest{Name: "默认"})
	require.NoError(t, err)

	feed, err := svc.Create(1, CreateFeedRequest{URL: ts.URL, CategoryID: cat.ID, UpdateIntervalMinutes: 60})
	require.NoError(t, err)
	assert.Equal(t, ts.URL, feed.URL)
	assert.Equal(t, "Test Feed", feed.Title)
	assert.NotNil(t, feed.CategoryID)
	assert.Equal(t, cat.ID, *feed.CategoryID)
	assert.Equal(t, 90, feed.ExpireDays) // 默认 90 天

	// 显式设置永不过期
	expire0 := 0
	feed0, err := svc.Create(1, CreateFeedRequest{URL: ts.URL + "/never", CategoryID: cat.ID, UpdateIntervalMinutes: 60, ExpireDays: &expire0})
	require.NoError(t, err)
	assert.Equal(t, 0, feed0.ExpireDays)

	// 代理不存在应失败
	proxyID := uint(999)
	_, err = svc.Create(1, CreateFeedRequest{URL: ts.URL + "/2", CategoryID: cat.ID, UpdateIntervalMinutes: 60, ProxyID: &proxyID})
	assert.Equal(t, "代理不存在", err.Error())
}

func TestFeedService_List(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	feeds, err := svc.List(1)
	require.NoError(t, err)
	assert.Empty(t, feeds)
}

func TestFeedService_GetByID_NotFound(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	_, err := svc.GetByID(1, 999)
	assert.ErrorIs(t, err, ErrFeedNotFound)
}

func TestFeedService_Delete_NotFound(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	err := svc.Delete(1, 999)
	assert.ErrorIs(t, err, ErrFeedNotFound)
}

func TestFeedService_Update_WithProxy(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)
	catSvc := NewCategoryService(db)
	proxySvc := NewProxyService(db)

	cat, err := catSvc.Create(1, CreateCategoryRequest{Name: "默认"})
	require.NoError(t, err)
	proxy, err := proxySvc.Create(1, CreateProxyRequest{Name: "代理", URL: "http://127.0.0.1:7890"})
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?><rss version="2.0"><channel><title>F</title></channel></rss>`)
	}))
	defer ts.Close()

	feed, err := svc.Create(1, CreateFeedRequest{URL: ts.URL, CategoryID: cat.ID, UpdateIntervalMinutes: 60})
	require.NoError(t, err)

	updated, err := svc.Update(1, feed.ID, UpdateFeedRequest{UpdateIntervalMinutes: 120, ProxyID: &proxy.ID})
	require.NoError(t, err)
	assert.Equal(t, 120, updated.UpdateIntervalMinutes)
	assert.NotNil(t, updated.ProxyID)
	assert.Equal(t, proxy.ID, *updated.ProxyID)

	// 更新过期时间
	expire30 := 30
	updated2, err := svc.Update(1, feed.ID, UpdateFeedRequest{UpdateIntervalMinutes: 120, ProxyID: &proxy.ID, ExpireDays: &expire30})
	require.NoError(t, err)
	assert.Equal(t, 30, updated2.ExpireDays)
}
