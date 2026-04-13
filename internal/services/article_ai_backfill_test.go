package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
)

func TestBackfillClassifyBatch(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "{\"category\":\"补漏\"}"}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "bf", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://bf.example/f", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("bfg"), GUIDRaw: "bfg",
		Title: "标题", Content: "正文",
		AIProcessStatus: models.AIProcessFailed, AILastError: "timeout",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillClassifyBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "补漏", out.AICategory)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
}

func TestBackfillTranslateBatch_translateOnly(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"TT","content_translated":"BB"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "bft", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://bf.example/t", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("bftg"), GUIDRaw: "bftg",
		Title: "标题", Content: "正文", AIProcessStatus: "",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillTranslateBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "TT", out.TitleTranslated)
	assert.Equal(t, "BB", out.ContentTranslated)
}

func TestBackfillClassifyBatch_skipsPending(t *testing.T) {
	db := setupArticleAIDB(t)
	user := models.User{Username: "p", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: "http://x/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://p.example/f", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("pp"), GUIDRaw: "pp",
		Title: "t", Content: "c", AIProcessStatus: models.AIProcessPending,
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillClassifyBatch(5, 0)
	assert.Equal(t, 0, ok)
	assert.Equal(t, 0, fail)
}
