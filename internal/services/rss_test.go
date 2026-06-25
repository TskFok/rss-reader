package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
)

func TestRSSService_FetchFeed_DoesNotPersistPendingBeforeAsyncWorkerStarts(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed</title>
    <item>
      <guid>recoverable-guid</guid>
      <title>Article</title>
      <link>https://example.com/a</link>
      <description>Body</description>
    </item>
  </channel>
</rss>`))
	}))
	defer ts.Close()

	user := models.User{Username: "rss", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	modelID := uint(1)
	feed := models.Feed{
		UserID: user.ID, URL: ts.URL, Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &modelID, AIClassifyEnabled: true,
	}
	require.NoError(t, db.Create(&feed).Error)

	rssSvc := NewRSSService(db)
	require.NoError(t, rssSvc.FetchFeed(&feed))

	var out models.Article
	require.NoError(t, db.First(&out, "guid_raw = ?", "recoverable-guid").Error)
	assert.Equal(t, "", out.AIProcessStatus)
}

func TestRSSService_existingArticleGUIDs(t *testing.T) {
	db := setupArticleAIDB(t)
	user := models.User{Username: "rss-batch", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com/feed", Title: "F", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)

	g1 := models.ArticleGUIDHash("guid-1")
	g2 := models.ArticleGUIDHash("guid-2")
	require.NoError(t, db.Create(&models.Article{
		FeedID: feed.ID, GUID: g1, GUIDRaw: "guid-1", Title: "a1",
	}).Error)

	rssSvc := NewRSSService(db)
	got, err := rssSvc.existingArticleGUIDs(feed.ID, []string{g1, g2, g1})
	require.NoError(t, err)
	assert.Len(t, got, 1)
	_, ok := got[g1]
	assert.True(t, ok)
	_, ok = got[g2]
	assert.False(t, ok)

	empty, err := rssSvc.existingArticleGUIDs(feed.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestRSSService_FetchFeed_SkipsExistingArticlesOnRefresh(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed</title>
    <item>
      <guid>existing-guid</guid>
      <title>Existing Article</title>
      <link>https://example.com/existing</link>
      <description>Body</description>
    </item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	user := models.User{Username: "rss-refresh", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: ts.URL, Title: "F", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)
	require.NoError(t, db.Create(&models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("existing-guid"), GUIDRaw: "existing-guid", Title: "Existing Article",
	}).Error)

	rssSvc := NewRSSService(db)
	require.NoError(t, rssSvc.FetchFeed(&feed))

	var count int64
	require.NoError(t, db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestRSSService_FetchFeed_InsertsOnlyNewArticles(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed</title>
    <item>
      <guid>old-guid</guid>
      <title>Old Article</title>
      <link>https://example.com/old</link>
    </item>
    <item>
      <guid>new-guid</guid>
      <title>New Article</title>
      <link>https://example.com/new</link>
    </item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	user := models.User{Username: "rss-mixed", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: ts.URL, Title: "F", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)
	require.NoError(t, db.Create(&models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("old-guid"), GUIDRaw: "old-guid", Title: "Old Article",
	}).Error)

	rssSvc := NewRSSService(db)
	require.NoError(t, rssSvc.FetchFeed(&feed))

	var article models.Article
	require.NoError(t, db.Where("feed_id = ? AND guid_raw = ?", feed.ID, "new-guid").First(&article).Error)
	assert.Equal(t, "New Article", article.Title)

	var count int64
	require.NoError(t, db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestRSSService_FetchFeed_SkipsDuplicateItemsInSameFeed(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed</title>
    <item>
      <guid>dup-guid</guid>
      <title>First</title>
      <link>https://example.com/dup-1</link>
    </item>
    <item>
      <guid>dup-guid</guid>
      <title>Second</title>
      <link>https://example.com/dup-2</link>
    </item>
  </channel>
</rss>`)
	}))
	defer ts.Close()

	user := models.User{Username: "rss-dup", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: ts.URL, Title: "F", UpdateIntervalMinutes: 60}
	require.NoError(t, db.Create(&feed).Error)

	rssSvc := NewRSSService(db)
	require.NoError(t, rssSvc.FetchFeed(&feed))

	var count int64
	require.NoError(t, db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	var article models.Article
	require.NoError(t, db.Where("feed_id = ?", feed.ID).First(&article).Error)
	assert.Equal(t, "First", article.Title)
}
