package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupArticleAIDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	// 与 feed_test 一致的全量迁移，避免 sqlite 下缺列
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.FeedCategory{}, &models.Proxy{}, &models.AIModel{}, &models.Feed{}, &models.Article{}, &models.UserArticle{},
	))
	return db
}

func TestFeedNeedsAIProcessing(t *testing.T) {
	assert.False(t, FeedNeedsAIProcessing(nil))
	mid := uint(1)
	assert.False(t, FeedNeedsAIProcessing(&models.Feed{AIModelID: &mid}))
	assert.True(t, FeedNeedsAIProcessing(&models.Feed{AIModelID: &mid, AIClassifyEnabled: true}))
	assert.False(t, FeedNeedsAIProcessing(&models.Feed{AIModelID: &mid, AITranslateEnabled: true, AITargetLanguage: ""}))
	assert.True(t, FeedNeedsAIProcessing(&models.Feed{AIModelID: &mid, AITranslateEnabled: true, AITargetLanguage: "en"}))
}

func TestExtractJSONObject(t *testing.T) {
	raw := "```json\n{\"category\":\"科技\"}\n```"
	assert.Equal(t, "{\"category\":\"科技\"}", extractJSONObject(raw))
}

func TestBuildClassifySystemPrompt_DomainBuckets(t *testing.T) {
	s := buildClassifySystemPrompt()
	assert.Contains(t, s, "财经")
	assert.Contains(t, s, "军事")
	assert.Contains(t, s, "归纳")
}

func TestArticleAIProcessor_run_ClassifyOnly(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/chat/completions")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "{\"category\":\"科技\"}"}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "u", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID:                user.ID,
		URL:                   "http://example.com/f",
		Title:                 "F",
		UpdateIntervalMinutes: 60,
		AIModelID:             &m.ID,
		AIClassifyEnabled:     true,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "标题", Content: "<p>正文</p>"}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	p.run(user.ID, feed, art.ID)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Equal(t, "科技", out.AICategory)
}

func TestArticleAIProcessor_run_TranslateOnly(t *testing.T) {
	db := setupArticleAIDB(t)
	var requestBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(raw)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"T","content_translated":"<p>译文<img src=\"https://cdn.example.com/a.jpg\"></p>"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "u2", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID:                user.ID,
		URL:                   "http://example.com/f2",
		Title:                 "F",
		UpdateIntervalMinutes: 60,
		AIModelID:             &m.ID,
		AITranslateEnabled:    true,
		AITargetLanguage:      "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID:  feed.ID,
		GUID:    models.ArticleGUIDHash("g2"),
		GUIDRaw: "g2",
		Title:   "标题",
		Content: `<p>正文<img src="https://cdn.example.com/a.jpg"></p>`,
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	p.run(user.ID, feed, art.ID)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Equal(t, "T", out.TitleTranslated)
	assert.Equal(t, `<p>译文<img src="https://cdn.example.com/a.jpg"></p>`, out.ContentTranslated)
	assert.Contains(t, requestBody, `Body HTML:\n<p>正文<img src=\"https://cdn.example.com/a.jpg\"></p>`)
	assert.Contains(t, requestBody, `Body plain text (for reference):\n正文`)
}

func TestBuildTranslateSystemPrompt_PreservesHTML(t *testing.T) {
	s := buildTranslateSystemPrompt("en", false)
	assert.Contains(t, s, "preserve the original HTML structure")
	assert.Contains(t, s, "Do not drop elements")
}

func TestArticleAIProcessor_run_ClassifyThenTranslate(t *testing.T) {
	db := setupArticleAIDB(t)
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		var content string
		if n == 1 {
			content = `{"category":"科技"}`
		} else {
			content = `{"category_translated":"Tech","title_translated":"T","content_translated":"B"}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": content}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "u3", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID:                user.ID,
		URL:                   "http://example.com/f3",
		Title:                 "F",
		UpdateIntervalMinutes: 60,
		AIModelID:             &m.ID,
		AIClassifyEnabled:     true,
		AITranslateEnabled:    true,
		AITargetLanguage:      "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g3"), GUIDRaw: "g3", Title: "标题", Content: "正文"}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	p.run(user.ID, feed, art.ID)

	assert.Equal(t, 2, n)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Equal(t, "科技", out.AICategory)
	assert.Equal(t, "Tech", out.AICategoryTranslated)
	assert.Equal(t, "T", out.TitleTranslated)
	assert.Equal(t, "B", out.ContentTranslated)
}

func TestArticleAIProcessor_ManualClassify(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "{\"category\":\"军事\"}"}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "mc", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/mc", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("mg"), GUIDRaw: "mg", Title: "标题", Content: "正文"}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	require.NoError(t, p.ManualClassify(user.ID, art.ID, nil))

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "军事", out.AICategory)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
}

func TestArticleAIProcessor_ManualClassify_AlreadyHasCategory(t *testing.T) {
	db := setupArticleAIDB(t)
	user := models.User{Username: "mc2", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: "http://localhost/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/mc2", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID,
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("mg2"), GUIDRaw: "mg2", Title: "标题",
		AICategory: "科技",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	assert.ErrorIs(t, p.ManualClassify(user.ID, art.ID, nil), ErrManualAIAlreadyClassified)
}

func TestArticleAIProcessor_ManualTranslate(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"T","content_translated":"B"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "mt", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/mt", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("mtg"), GUIDRaw: "mtg", Title: "标题", Content: "正文"}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	require.NoError(t, p.ManualTranslate(user.ID, art.ID, nil, ""))

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, "T", out.TitleTranslated)
	assert.Equal(t, "B", out.ContentTranslated)
}

func TestArticleAIProcessor_ManualTranslate_InvalidOverrideLang(t *testing.T) {
	db := setupArticleAIDB(t)
	user := models.User{Username: "inv", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: "http://x/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/inv", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("inv"), GUIDRaw: "inv", Title: "t", Content: "c"}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	assert.ErrorIs(t, p.ManualTranslate(user.ID, art.ID, nil, "not-a-lang"), ErrManualAIInvalidTargetLang)
}
