package services

import (
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
