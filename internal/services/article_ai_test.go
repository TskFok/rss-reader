package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

	var sent chatCompletionsRequest
	require.NoError(t, json.Unmarshal([]byte(requestBody), &sent))
	require.Len(t, sent.Messages, 2)
	assert.Contains(t, sent.Messages[1].Content, "Body HTML:\n<p>正文<img src=\"https://cdn.example.com/a.jpg\"></p>")
	assert.Contains(t, sent.Messages[1].Content, "Body plain text (for reference):\n正文")
}

func TestBuildTranslateSystemPrompt_PreservesHTML(t *testing.T) {
	s := buildTranslateSystemPrompt("en", false)
	assert.Contains(t, s, "preserve the original HTML structure")
	assert.Contains(t, s, "Do not drop elements")
}

func TestArticleAIProcessor_run_ClassifyThenTranslate_UsesSingleModelCall(t *testing.T) {
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
				{"message": map[string]string{"content": `{"category":"科技","category_translated":"Tech","title_translated":"T","content_translated":"B"}`}},
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

	assert.Equal(t, 1, n)

	var sent chatCompletionsRequest
	require.NoError(t, json.Unmarshal([]byte(requestBody), &sent))
	require.Len(t, sent.Messages, 2)
	assert.Contains(t, sent.Messages[0].Content, `"category"`)
	assert.Contains(t, sent.Messages[0].Content, `"category_translated"`)
	assert.Contains(t, sent.Messages[0].Content, `"title_translated"`)
	assert.Contains(t, sent.Messages[0].Content, `"content_translated"`)

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Equal(t, "科技", out.AICategory)
	assert.Equal(t, "Tech", out.AICategoryTranslated)
	assert.Equal(t, "T", out.TitleTranslated)
	assert.Equal(t, "B", out.ContentTranslated)
}

func TestArticleAIProcessor_EnqueueLimitsAutomaticConcurrency(t *testing.T) {
	db := setupArticleAIDB(t)
	var mu sync.Mutex
	inFlight := 0
	maxInFlight := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()

		time.Sleep(80 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"T","content_translated":"B"}`}},
			},
		})

		mu.Lock()
		inFlight--
		mu.Unlock()
	}))
	defer ts.Close()

	user := models.User{Username: "auto-concurrency", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/auto-concurrency", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	ids := make([]uint, 0, 3)
	for i := 0; i < 3; i++ {
		art := models.Article{
			FeedID: feed.ID, GUID: models.ArticleGUIDHash("auto-concurrency-" + strconv.Itoa(i)),
			GUIDRaw: "auto-concurrency", Title: "标题", Content: "正文",
		}
		require.NoError(t, db.Create(&art).Error)
		ids = append(ids, art.ID)
	}

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	defer p.Stop()
	for _, id := range ids {
		p.Enqueue(&feed, id)
	}

	require.Eventually(t, func() bool {
		var n int64
		require.NoError(t, db.Model(&models.Article{}).
			Where("feed_id = ? AND content_translated <> ''", feed.ID).
			Count(&n).Error)
		return n == int64(len(ids))
	}, 5*time.Second, 20*time.Millisecond)

	mu.Lock()
	got := maxInFlight
	mu.Unlock()
	assert.Equal(t, 1, got)
}

func TestArticleAIProcessor_EnqueueRateLimitsAutomaticJobs(t *testing.T) {
	db := setupArticleAIDB(t)
	var mu sync.Mutex
	var starts []time.Time
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"T","content_translated":"B"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "auto-rate", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/auto-rate", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	ids := make([]uint, 0, 2)
	for i := 0; i < 2; i++ {
		art := models.Article{
			FeedID: feed.ID, GUID: models.ArticleGUIDHash("auto-rate-" + strconv.Itoa(i)),
			GUIDRaw: "auto-rate", Title: "标题", Content: "正文",
		}
		require.NoError(t, db.Create(&art).Error)
		ids = append(ids, art.ID)
	}

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	defer p.Stop()
	for _, id := range ids {
		p.Enqueue(&feed, id)
	}

	require.Eventually(t, func() bool {
		var n int64
		require.NoError(t, db.Model(&models.Article{}).
			Where("feed_id = ? AND content_translated <> ''", feed.ID).
			Count(&n).Error)
		return n == int64(len(ids))
	}, 5*time.Second, 20*time.Millisecond)

	mu.Lock()
	gotStarts := append([]time.Time(nil), starts...)
	mu.Unlock()
	require.Len(t, gotStarts, len(ids))
	gap := gotStarts[1].Sub(gotStarts[0])
	assert.GreaterOrEqual(t, gap, 500*time.Millisecond)
}

func TestArticleAIProcessor_EnqueueClassifyThenTranslateUsesOneModelCall(t *testing.T) {
	db := setupArticleAIDB(t)
	var mu sync.Mutex
	var starts []time.Time
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"category":"科技","category_translated":"Tech","title_translated":"T","content_translated":"B"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "auto-rate-pipeline", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/auto-rate-pipeline", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("auto-rate-pipeline"),
		GUIDRaw: "auto-rate-pipeline", Title: "标题", Content: "正文",
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	defer p.Stop()
	p.Enqueue(&feed, art.ID)

	require.Eventually(t, func() bool {
		var out models.Article
		require.NoError(t, db.First(&out, art.ID).Error)
		return strings.TrimSpace(out.ContentTranslated) != ""
	}, 5*time.Second, 20*time.Millisecond)

	mu.Lock()
	gotStarts := append([]time.Time(nil), starts...)
	mu.Unlock()
	require.Len(t, gotStarts, 1)
}

func TestArticleAIProcessor_run_ClassifyAndTranslateCompletesSingleCallWhenFeedDeletedDuringCall(t *testing.T) {
	db := setupArticleAIDB(t)
	var feed models.Feed
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			require.NoError(t, db.Delete(&feed).Error)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"category":"科技","category_translated":"Tech","title_translated":"T","content_translated":"B"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "early-return", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed = models.Feed{
		UserID: user.ID, URL: "http://example.com/early-return", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AIClassifyEnabled: true, AITranslateEnabled: true, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("early-return"), GUIDRaw: "early-return",
		Title: "标题", Content: "正文", AIProcessStatus: models.AIProcessPending,
	}
	require.NoError(t, db.Create(&art).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	p.run(user.ID, feed, art.ID)

	assert.Equal(t, 1, n)
	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Empty(t, out.AILastError)
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

func TestArticleAIProcessor_ManualTranslate_AllowsStalePending(t *testing.T) {
	db := setupArticleAIDB(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title_translated":"Recovered","content_translated":"Body"}`}},
			},
		})
	}))
	defer ts.Close()

	user := models.User{Username: "stale-manual", PasswordHash: "x"}
	require.NoError(t, db.Create(&user).Error)
	m := models.AIModel{UserID: user.ID, Name: "m", BaseURL: ts.URL + "/v1"}
	require.NoError(t, db.Create(&m).Error)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com/stale-manual", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &m.ID, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	art := models.Article{
		FeedID: feed.ID, GUID: models.ArticleGUIDHash("stale-manual"), GUIDRaw: "stale-manual",
		Title: "t", Content: "c", AIProcessStatus: models.AIProcessPending,
	}
	require.NoError(t, db.Create(&art).Error)
	require.NoError(t, db.Model(&models.Article{}).Where("id = ?", art.ID).
		Update("updated_at", time.Now().Add(-31*time.Minute)).Error)

	p := NewArticleAIProcessor(db, NewAIModelService(db))
	require.NoError(t, p.ManualTranslate(user.ID, art.ID, nil, ""))

	var out models.Article
	require.NoError(t, db.First(&out, art.ID).Error)
	assert.Equal(t, models.AIProcessDone, out.AIProcessStatus)
	assert.Equal(t, "Recovered", out.TitleTranslated)
	assert.Equal(t, "Body", out.ContentTranslated)
}
