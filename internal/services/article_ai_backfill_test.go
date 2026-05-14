package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestBackfillClassifyBatch_ClassifyAndTranslateUsesSingleModelCall(t *testing.T) {
	db := setupArticleAIDB(t)
	var n int
	var requestBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(raw)
		n++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"category":"科技","category_translated":"Tech","title_translated":"TT","content_translated":"BB"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "bf-combined", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://bf.example/combined", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("bf-combined"), GUIDRaw: "bf-combined",
		Title: "标题", Content: "正文",
		AIProcessStatus: models.AIProcessFailed, AILastError: "timeout",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillClassifyBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 1, n)

	var sent chatCompletionsRequest
	require.NoError(t, json.Unmarshal([]byte(requestBody), &sent))
	require.Len(t, sent.Messages, 2)
	assert.Contains(t, sent.Messages[0].Content, `"category"`)
	assert.Contains(t, sent.Messages[0].Content, `"content_translated"`)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "科技", out.AICategory)
	assert.Equal(t, "Tech", out.AICategoryTranslated)
	assert.Equal(t, "TT", out.TitleTranslated)
	assert.Equal(t, "BB", out.ContentTranslated)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
}

func TestBackfillClassifyBatch_ClassifyAndTranslateWhenOnlyTitleTranslated(t *testing.T) {
	db := setupArticleAIDB(t)
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"category":"科技","category_translated":"Tech","title_translated":"TT","content_translated":"BB"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "bf-title-only-combined", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://bf.example/title-only-combined", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("bf-title-only-combined"), GUIDRaw: "bf-title-only-combined",
		Title: "标题", Content: "正文", TitleTranslated: "Old title",
		AIProcessStatus: models.AIProcessFailed, AILastError: "timeout",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillClassifyBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 1, n)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "科技", out.AICategory)
	assert.Equal(t, "Tech", out.AICategoryTranslated)
	assert.Equal(t, "TT", out.TitleTranslated)
	assert.Equal(t, "BB", out.ContentTranslated)
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

func TestBackfillTranslateBatch_RetriesWhenOnlyTitleTranslated(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"New title","content_translated":"New body"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "bft-title-only", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://bf.example/title-only", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("bft-title-only"), GUIDRaw: "bft-title-only",
		Title: "标题", Content: "正文", TitleTranslated: "Old title", ContentTranslated: "",
		AIProcessStatus: models.AIProcessDone,
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillTranslateBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "New title", out.TitleTranslated)
	assert.Equal(t, "New body", out.ContentTranslated)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
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

func TestBackfillClassifyBatch_recoversStalePending(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "{\"category\":\"恢复\"}"}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "stale-p", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://stale.example/f", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("stale-p"), GUIDRaw: "stale-p",
		Title: "t", Content: "c", AIProcessStatus: models.AIProcessPending,
	}
	require.NoError(t, db.Create(&art).Error)
	require.NoError(t, db.Model(&models.Article{}).Where("id = ?", art.ID).
		Update("updated_at", time.Now().Add(-31*time.Minute)).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	ok, fail := p.BackfillClassifyBatch(5, 0)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, fail)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "恢复", out.AICategory)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
}
