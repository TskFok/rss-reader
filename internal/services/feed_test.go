package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupFeedDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Proxy{}, &models.AIModel{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
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

	var art models.Article
	require.NoError(t, db.Where("feed_id = ?", feed.ID).First(&art).Error)
	assert.Equal(t, "hello", art.GUIDRaw)
	assert.Equal(t, models.ArticleGUIDHash("hello"), art.GUID)

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

func TestFeedService_Refresh_FetchesArticlesForOwnedFeed(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Manual Refresh Feed</title>
    <link>http://example.com/</link>
    <description>test</description>
    <item>
      <title>Manual Refresh Article</title>
      <link>http://example.com/manual-refresh</link>
      <guid>manual-refresh-guid</guid>
    </item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	feed := models.Feed{UserID: 1, URL: ts.URL, Title: "Manual", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)

	refreshed, err := svc.Refresh(1, feed.ID)
	require.NoError(t, err)
	require.NotNil(t, refreshed.LastFetchedAt)

	var article models.Article
	require.NoError(t, db.Where("feed_id = ? AND guid_raw = ?", feed.ID, "manual-refresh-guid").First(&article).Error)
	assert.Equal(t, "Manual Refresh Article", article.Title)
}

func TestFeedService_Refresh_RejectsOtherUsersFeed(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	feed := models.Feed{UserID: 2, URL: "http://example.com/feed.xml", Title: "Other", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)

	_, err := svc.Refresh(1, feed.ID)
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

func TestFeedService_Update_Category(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)
	catSvc := NewCategoryService(db)

	cat1, err := catSvc.Create(1, CreateCategoryRequest{Name: "分类1"})
	require.NoError(t, err)
	cat2, err := catSvc.Create(1, CreateCategoryRequest{Name: "分类2"})
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?><rss version="2.0"><channel><title>F</title></channel></rss>`)
	}))
	defer ts.Close()

	feed, err := svc.Create(1, CreateFeedRequest{URL: ts.URL, CategoryID: cat1.ID, UpdateIntervalMinutes: 60})
	require.NoError(t, err)
	assert.Equal(t, cat1.ID, *feed.CategoryID)

	// 更新分类
	updated, err := svc.Update(1, feed.ID, UpdateFeedRequest{
		UpdateIntervalMinutes: 60,
		CategoryID:            &cat2.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, cat2.ID, *updated.CategoryID)

	// 分类不存在应失败
	badCatID := uint(999)
	_, err = svc.Update(1, feed.ID, UpdateFeedRequest{
		UpdateIntervalMinutes: 60,
		CategoryID:            &badCatID,
	})
	assert.Equal(t, "分类不存在", err.Error())

	// 清空分类（category_id=0）
	zero := uint(0)
	updated2, err := svc.Update(1, feed.ID, UpdateFeedRequest{
		UpdateIntervalMinutes: 60,
		CategoryID:            &zero,
	})
	require.NoError(t, err)
	assert.Nil(t, updated2.CategoryID)
}

func TestFeedService_Update_URL(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	feed := models.Feed{UserID: 1, URL: "http://example.com/feed.xml", Title: "F", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)

	nextURL := "https://example.com/changed.xml"
	updated, err := svc.Update(1, feed.ID, UpdateFeedRequest{
		URL:                   &nextURL,
		UpdateIntervalMinutes: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, nextURL, updated.URL)

	var stored models.Feed
	require.NoError(t, db.First(&stored, feed.ID).Error)
	assert.Equal(t, nextURL, stored.URL)
}

func TestFeedService_Create_AIValidation(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)
	catSvc := NewCategoryService(db)
	cat, err := catSvc.Create(1, CreateCategoryRequest{Name: "默认"})
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?><rss version="2.0"><channel><title>F</title></channel></rss>`)
	}))
	defer ts.Close()

	_, err = svc.Create(1, CreateFeedRequest{URL: ts.URL + "/a", CategoryID: cat.ID, UpdateIntervalMinutes: 60, AIClassifyEnabled: true})
	assert.Equal(t, "开启 AI 分类或翻译时需选择模型", err.Error())

	m := models.AIModel{UserID: 1, Name: "m", BaseURL: "http://localhost/v1"}
	require.NoError(t, db.Create(&m).Error)
	_, err = svc.Create(1, CreateFeedRequest{
		URL: ts.URL + "/b", CategoryID: cat.ID, UpdateIntervalMinutes: 60,
		AITranslateEnabled: true, AIModelID: &m.ID,
	})
	assert.Equal(t, "开启 AI 翻译时需填写目标语言", err.Error())

	feed, err := svc.Create(1, CreateFeedRequest{
		URL: ts.URL + "/c", CategoryID: cat.ID, UpdateIntervalMinutes: 60,
		AIClassifyEnabled: true, AIModelID: &m.ID,
	})
	require.NoError(t, err)
	assert.True(t, feed.AIClassifyEnabled)
	require.NotNil(t, feed.AIModelID)
	assert.Equal(t, m.ID, *feed.AIModelID)
}
