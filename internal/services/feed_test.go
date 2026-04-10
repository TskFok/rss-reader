package services

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupFeedDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.FeedCategory{},
		&models.Proxy{},
		&models.Feed{},
		&models.FeedAISetting{},
		&models.FeedAIClassificationJob{},
		&models.FeedAICategory{},
		&models.AIModel{},
		&models.Article{},
		&models.ArticleAIMetadata{},
		&models.ArticleAIMetadataJob{},
		&models.UserArticle{},
		&models.ArticleCluster{},
	))
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

func TestRSSService_FetchFeed_EnqueuesMetadataJobsOnly(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item><title>A1</title><link>http://e/a1</link><guid>a1</guid><description>d1</description></item>
    <item><title>A2</title><link>http://e/a2</link><guid>a2</guid><description>d2</description></item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	feed.URL = ts.URL
	require.NoError(t, rss.FetchFeed(&feed))

	var articleCount int64
	require.NoError(t, db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&articleCount).Error)
	assert.Equal(t, int64(2), articleCount)

	var metaCount int64
	require.NoError(t, db.Model(&models.ArticleAIMetadata{}).Where("user_id = ?", user.ID).Count(&metaCount).Error)
	assert.Equal(t, int64(0), metaCount)

	var jobCount int64
	require.NoError(t, db.Model(&models.ArticleAIMetadataJob{}).Where("user_id = ? AND status = ?", user.ID, "pending").Count(&jobCount).Error)
	assert.Equal(t, int64(2), jobCount)
}

func TestFeedService_List(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)
	svc := NewFeedService(db, rss)

	feeds, err := svc.List(1)
	require.NoError(t, err)
	assert.Empty(t, feeds)
}

func TestRSSService_FetchFeed_FeedAIMetadata_OnlyWhenConfigured(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)

	// 模拟 OpenAI 兼容接口，返回严格 JSON
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"summary\":\"本次新增主要围绕数据库与AI\",\"categories\":[\"数据库\",\"AI\"]}"}}]}`)
	}))
	defer aiServer.Close()

	m := models.AIModel{UserID: user.ID, Name: "test-model", BaseURL: aiServer.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	// RSS 源：两篇新文章
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item><title>A1</title><link>http://e/a1</link><guid>a1</guid><description>mysql</description></item>
    <item><title>A2</title><link>http://e/a2</link><guid>a2</guid><description>ai agent</description></item>
  </channel>
</rss>`)
	}))
	defer ts.Close()
	feed.URL = ts.URL

	// 未配置：不应写入 setting 结果与分类
	require.NoError(t, rss.FetchFeed(&feed))
	var st0 models.FeedAISetting
	err := db.Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).First(&st0).Error
	require.Error(t, err)
	var catCount0 int64
	require.NoError(t, db.Model(&models.FeedAICategory{}).Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).Count(&catCount0).Error)
	assert.Equal(t, int64(0), catCount0)

	// 配置 AI：创建 setting（enabled + model）
	setting := models.FeedAISetting{UserID: user.ID, FeedID: feed.ID, Enabled: true, AIModelID: &m.ID}
	require.NoError(t, db.Create(&setting).Error)

	// 再次抓取（换 guid，确保有新文章触发）
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item><title>B1</title><link>http://e/b1</link><guid>b1</guid><description>mysql</description></item>
  </channel>
</rss>`)
	}))
	defer ts2.Close()
	feed.URL = ts2.URL
	require.NoError(t, rss.FetchFeed(&feed))

	var pendingJobs int64
	require.NoError(t, db.Model(&models.FeedAIClassificationJob{}).
		Where("user_id = ? AND feed_id = ? AND status = ?", user.ID, feed.ID, "pending").
		Count(&pendingJobs).Error)
	assert.Equal(t, int64(1), pendingJobs)

	feedClassJobs := NewFeedAIClassificationJobService(db)
	_, err = feedClassJobs.ProcessPending(10)
	require.NoError(t, err)

	var finJob models.FeedAIClassificationJob
	require.NoError(t, db.Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).First(&finJob).Error)
	assert.Equal(t, "done", finJob.Status)

	var st models.FeedAISetting
	require.NoError(t, db.Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).First(&st).Error)
	assert.NotEmpty(t, st.Summary)
	assert.Contains(t, st.CategoriesJSON, "数据库")

	var catCount int64
	require.NoError(t, db.Model(&models.FeedAICategory{}).Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).Count(&catCount).Error)
	assert.Equal(t, int64(2), catCount)
}

func TestRSSService_FetchFeed_CustomClassifierPromptInRequest(t *testing.T) {
	db := setupFeedDB(t)
	rss := NewRSSService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)

	var lastBody string
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		lastBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"summary\":\"s\",\"categories\":[\"x\"]}"}}]}`)
	}))
	defer aiServer.Close()

	m := models.AIModel{UserID: user.ID, Name: "test-model", BaseURL: aiServer.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	custom := "CUSTOM_PROMPT_MARKER_FOR_TEST"
	setting := models.FeedAISetting{
		UserID: user.ID, FeedID: feed.ID, Enabled: true, AIModelID: &m.ID,
		ClassifierPrompt: custom,
	}
	require.NoError(t, db.Create(&setting).Error)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>T</title>
<item><title>N</title><link>http://e/n</link><guid>n1</guid><description>d</description></item>
</channel></rss>`)
	}))
	defer ts.Close()
	feed.URL = ts.URL
	require.NoError(t, rss.FetchFeed(&feed))
	_, err := NewFeedAIClassificationJobService(db).ProcessPending(10)
	require.NoError(t, err)
	assert.True(t, strings.Contains(lastBody, custom), "请求体应包含自定义 prompt")
}

func TestFeedService_Update_ClassifierPrompt(t *testing.T) {
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

	feed, err := svc.Create(1, CreateFeedRequest{URL: ts.URL, CategoryID: cat.ID, UpdateIntervalMinutes: 60})
	require.NoError(t, err)

	p1 := "my custom instructions"
	updated, err := svc.Update(1, feed.ID, UpdateFeedRequest{
		UpdateIntervalMinutes: 60,
		AIClassifierPrompt:    &p1,
	})
	require.NoError(t, err)
	assert.Equal(t, p1, updated.AIClassifierPrompt)

	p2 := ""
	updated2, err := svc.Update(1, feed.ID, UpdateFeedRequest{
		UpdateIntervalMinutes: 60,
		AIClassifierPrompt:    &p2,
	})
	require.NoError(t, err)
	assert.Equal(t, "", updated2.AIClassifierPrompt)
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
