package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupSummaryRunnerDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.FeedCategory{},
		&models.Feed{},
		&models.Article{},
		&models.UserArticle{},
		&models.Proxy{},
		&models.AIModel{},
		&models.AISummaryHistory{},
	))
	return db
}

func TestRunDailySummaryForYesterday_CreatesHistoriesUntilEmpty(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	// mock AI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"这是中文总结内容。"}}]}`))
	}))
	defer server.Close()

	u := models.User{Username: "u", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "test-model", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	// create two articles yesterday (Shanghai date)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	t2 := time.Date(y.Year(), y.Month(), y.Day(), 12, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "a2", Content: "c2", PublishedAt: &t2}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	// page_size=1 -> should generate 2 histories then stop at empty page
	err = RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, nil, nil)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.AISummaryHistory{}).Where("user_id = ?", u.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// mockFeishuBotClient 用于测试的飞书机器人客户端，记录发送调用
type mockFeishuBotClient struct {
	sendCalls    []struct{ webhook, title, content string }
	apiSendCalls []struct{ appID, receiveID, title, content string }
	sendErr      error
}

func (m *mockFeishuBotClient) SendText(webhook string, title string, content string) error {
	m.sendCalls = append(m.sendCalls, struct{ webhook, title, content string }{webhook, title, content})
	return m.sendErr
}

func (m *mockFeishuBotClient) SendViaAPI(appID, appSecret, receiveIDType, receiveID, title, content string) error {
	m.apiSendCalls = append(m.apiSendCalls, struct{ appID, receiveID, title, content string }{appID, receiveID, title, content})
	return m.sendErr
}

func (m *mockFeishuBotClient) SendToUserByOpenID(openID string, title string, content string) error {
	m.apiSendCalls = append(m.apiSendCalls, struct{ appID, receiveID, title, content string }{"", openID, title, content})
	return m.sendErr
}

func TestRunDailySummaryForYesterday_SendsFeishuAlertOnAIFailure(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	// AI 返回错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := models.User{
		Username:         "alert-user",
		PasswordHash:     "h",
		Status:          models.UserStatusActive,
		FeishuBotWebhook: "https://open.feishu.cn/webhook/test",
	}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "my-model", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	mockBot := &mockFeishuBotClient{}
	err = RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, mockBot, db)
	require.Error(t, err)

	// 验证飞书告警被调用
	require.Len(t, mockBot.sendCalls, 1)
	call := mockBot.sendCalls[0]
	assert.Equal(t, "https://open.feishu.cn/webhook/test", call.webhook)
	assert.Equal(t, "[RSS Reader 定时总结失败告警]", call.title)
	assert.Contains(t, call.content, "用户：alert-user")
	assert.Contains(t, call.content, "模型：my-model")
	assert.Contains(t, call.content, "页码：1")
	assert.Contains(t, call.content, "文章数：1")
}

func TestRunDailySummaryForYesterday_SendsFeishuAlertViaAPIOnAIFailure(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	feishuOpenID := "open_xxx"
	u := models.User{
		Username:         "api-user",
		PasswordHash:     "h",
		Status:           models.UserStatusActive,
		FeishuNotifyType: "api",
		FeishuID:         &feishuOpenID,
	}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "api-model", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	mockBot := &mockFeishuBotClient{}
	_ = RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, mockBot, db)

	require.Len(t, mockBot.apiSendCalls, 1)
	call := mockBot.apiSendCalls[0]
	assert.Equal(t, "open_xxx", call.receiveID)
	assert.Equal(t, "[RSS Reader 定时总结失败告警]", call.title)
	assert.Contains(t, call.content, "用户：api-user")
}

func TestRunDailySummaryForYesterday_NoFeishuAlertWhenWebhookEmpty(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := models.User{Username: "no-webhook", PasswordHash: "h", Status: models.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "m", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	mockBot := &mockFeishuBotClient{}
	_ = RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, mockBot, db)
	assert.Len(t, mockBot.sendCalls, 0)
}

func TestRunDailySummaryForYesterday_NoFeishuAlertWhenFeishuBotNil(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := models.User{
		Username:         "has-webhook",
		PasswordHash:     "h",
		Status:           models.UserStatusActive,
		FeishuBotWebhook: "https://open.feishu.cn/webhook/test",
	}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "m", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	// feishuBot 为 nil，即使配置了 Webhook 也不发送
	err := RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, nil, db)
	require.Error(t, err)
	// 无 mock 可验证，主要确保传 nil 不 panic
}

func TestRunDailySummaryForYesterday_FeishuSendFailureDoesNotAffectMainFlow(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := models.User{
		Username:         "alert-user",
		PasswordHash:     "h",
		Status:           models.UserStatusActive,
		FeishuBotWebhook: "https://open.feishu.cn/webhook/test",
	}
	require.NoError(t, db.Create(&u).Error)
	m := models.AIModel{UserID: u.ID, Name: "my-model", BaseURL: server.URL}
	require.NoError(t, db.Create(&m).Error)

	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	mockBot := &mockFeishuBotClient{sendErr: assert.AnError}
	err := RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, m.ID, []uint{feed.ID}, 1, "desc", now, loc, mockBot, db)
	require.Error(t, err)
	// 主流程应返回 AI 错误，而非飞书错误
	assert.Contains(t, err.Error(), "500") // AI 服务返回 500
	// 飞书告警仍被尝试发送
	require.Len(t, mockBot.sendCalls, 1)
}

func TestTruncateString(t *testing.T) {
	// 短于 maxRunes 不截断
	assert.Equal(t, "short", truncateString("short", 500))
	assert.Equal(t, "", truncateString("", 500))

	// 等于 maxRunes 不截断
	s500 := strings.Repeat("a", 500)
	assert.Equal(t, s500, truncateString(s500, 500))

	// 超过 maxRunes 截断并加 "..."
	s600 := strings.Repeat("a", 600)
	result := truncateString(s600, 500)
	assert.Len(t, result, 503) // 500 + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
	assert.Equal(t, s500+"...", result)

	// 中文等多字节字符按 rune 截断
	chinese := strings.Repeat("中", 600)
	resultCn := truncateString(chinese, 500)
	assert.Equal(t, 500+3, len([]rune(resultCn)))
	assert.True(t, strings.HasSuffix(resultCn, "..."))
}

func TestRunDailySummaryForYesterday_ModelNotFoundShowsUnknown(t *testing.T) {
	db := setupSummaryRunnerDB(t)
	articleSvc := NewArticleService(db)
	historySvc := NewSummaryHistoryService(db)
	aiModelSvc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := models.User{
		Username:         "alert-user",
		PasswordHash:     "h",
		Status:           models.UserStatusActive,
		FeishuBotWebhook: "https://open.feishu.cn/webhook/test",
	}
	require.NoError(t, db.Create(&u).Error)
	// 不创建 AIModel，使用不存在的 aiModelID=999
	feed := models.Feed{UserID: u.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	y := now.AddDate(0, 0, -1)
	t1 := time.Date(y.Year(), y.Month(), y.Day(), 9, 0, 0, 0, loc)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1", Content: "c1", PublishedAt: &t1}
	require.NoError(t, db.Create(&a1).Error)

	mockBot := &mockFeishuBotClient{}
	// aiModelID=999 不存在，Summarize 会失败
	err := RunDailySummaryForYesterday(u.ID, aiModelSvc, articleSvc, historySvc, 999, []uint{feed.ID}, 1, "desc", now, loc, mockBot, db)
	require.Error(t, err)

	require.Len(t, mockBot.sendCalls, 1)
	call := mockBot.sendCalls[0]
	assert.Contains(t, call.content, "(未知)")
	assert.Contains(t, call.content, "ID: 999")
}

