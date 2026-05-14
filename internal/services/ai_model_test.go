package services

import (
	"encoding/json"
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

func setupAIModelDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AIModel{}))
	return db
}

func TestAIModelService_CRUD(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	// create
	m1, err := svc.Create(1, CreateAIModelRequest{
		Name:    "gpt-4o-mini",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test",
	})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o-mini", m1.Name)
	assert.Equal(t, "https://api.openai.com/v1", m1.BaseURL)
	assert.Equal(t, "sk-test", m1.APIKey)

	// list
	items, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, m1.ID, items[0].ID)

	// update
	updated, err := svc.Update(1, m1.ID, UpdateAIModelRequest{
		Name:    "gpt-4o",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  nil, // 不修改密钥
	})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", updated.Name)
	assert.Equal(t, "sk-test", updated.APIKey) // 保持不变

	// update with new api_key
	newKey := "sk-new"
	updated2, err := svc.Update(1, m1.ID, UpdateAIModelRequest{
		Name:    "gpt-4o",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  &newKey,
	})
	require.NoError(t, err)
	assert.Equal(t, "sk-new", updated2.APIKey)

	// create another for different user
	m2, err := svc.Create(2, CreateAIModelRequest{
		Name:    "ollama",
		BaseURL: "http://localhost:11434/v1",
	})
	require.NoError(t, err)
	assert.Equal(t, "ollama", m2.Name)
	assert.Equal(t, "", m2.APIKey)

	// list only returns own user's models
	items1, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, items1, 1)

	items2, err := svc.List(2)
	require.NoError(t, err)
	require.Len(t, items2, 1)

	// delete
	err = svc.Delete(1, m1.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(1, m1.ID)
	assert.ErrorIs(t, err, ErrAIModelNotFound)

	// delete other user's model should fail
	err = svc.Delete(1, m2.ID)
	assert.ErrorIs(t, err, ErrAIModelNotFound)
}

func TestAIModelService_EmptyBaseURL(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	_, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test",
		BaseURL: "   ",
	})
	assert.Error(t, err)

	m, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test",
		BaseURL: "https://api.example.com/v1",
	})
	require.NoError(t, err)

	_, err = svc.Update(1, m.ID, UpdateAIModelRequest{
		Name:    "test",
		BaseURL: "",
	})
	assert.Error(t, err)
}

func TestAIModelService_Test(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	// mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test-model",
		BaseURL: server.URL,
		APIKey:  "sk-test",
	})
	require.NoError(t, err)

	err = svc.Test(1, m.ID)
	require.NoError(t, err)
}

func TestAIModelService_Reorder(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	m1, _ := svc.Create(1, CreateAIModelRequest{Name: "a", BaseURL: "https://a.com/v1"})
	m2, _ := svc.Create(1, CreateAIModelRequest{Name: "b", BaseURL: "https://b.com/v1"})
	m3, _ := svc.Create(1, CreateAIModelRequest{Name: "c", BaseURL: "https://c.com/v1"})

	items, _ := svc.List(1)
	require.Len(t, items, 3)
	assert.Equal(t, m1.ID, items[0].ID)
	assert.Equal(t, m2.ID, items[1].ID)
	assert.Equal(t, m3.ID, items[2].ID)

	err := svc.Reorder(1, []uint{m3.ID, m1.ID, m2.ID})
	require.NoError(t, err)

	items, _ = svc.List(1)
	require.Len(t, items, 3)
	assert.Equal(t, m3.ID, items[0].ID)
	assert.Equal(t, m1.ID, items[1].ID)
	assert.Equal(t, m2.ID, items[2].ID)
}

func TestAIModelService_Test_Fail(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test-model",
		BaseURL: server.URL,
		APIKey:  "sk-invalid",
	})
	require.NoError(t, err)

	err = svc.Test(1, m.ID)
	assert.Error(t, err)
}

func TestAIModelService_Summarize(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"这是中文总结内容。"}}]}`))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test-model",
		BaseURL: server.URL,
		APIKey:  "sk-test",
	})
	require.NoError(t, err)

	articles := []ArticleForSummary{
		{Title: "文章1", Content: "内容1", FeedTitle: "订阅A", PublishedAt: "2025-03-01"},
		{Title: "文章2", Content: "内容2", FeedTitle: "订阅B", PublishedAt: "2025-03-02"},
	}
	summary, err := svc.Summarize(1, m.ID, articles, nil)
	require.NoError(t, err)
	assert.Equal(t, "这是中文总结内容。", summary)
}

func TestAIModelService_Summarize_EmptyArticles(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	_, err := svc.Summarize(1, 1, nil, nil)
	assert.Error(t, err)

	_, err = svc.Summarize(1, 1, []ArticleForSummary{}, nil)
	assert.Error(t, err)
}

func TestAIModelService_Summarize_ModelNotFound(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	articles := []ArticleForSummary{{Title: "a", Content: "b", FeedTitle: "f", PublishedAt: "2025-03-01"}}
	_, err := svc.Summarize(1, 999, articles, nil)
	assert.ErrorIs(t, err, ErrAIModelNotFound)
}

func TestAIModelService_ChatCompletionText_UsesBackupModelWhenPrimaryFails(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	var primaryCalls int
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalls++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer primary.Close()

	var backupCalls int
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupCalls++
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-backup", r.Header.Get("Authorization"))
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "backup-model", req.Model)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"备用模型结果"}}]}`))
	}))
	defer backup.Close()

	backupModel, err := svc.Create(1, CreateAIModelRequest{
		Name:    "backup-model",
		BaseURL: backup.URL,
		APIKey:  "sk-backup",
	})
	require.NoError(t, err)
	primaryModel, err := svc.Create(1, CreateAIModelRequest{
		Name:          "primary-model",
		BaseURL:       primary.URL,
		BackupModelID: &backupModel.ID,
	})
	require.NoError(t, err)

	got, err := svc.ChatCompletionText(1, primaryModel.ID, 100, []chatMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "备用模型结果", got)
	assert.Equal(t, 1, primaryCalls)
	assert.Equal(t, 1, backupCalls)
}

func TestAIModelService_Summarize_UsesBackupModelWhenPrimaryFails(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer primary.Close()

	var backupCalls int
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupCalls++
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "backup-summary", req.Model)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"备用总结"}}]}`))
	}))
	defer backup.Close()

	backupModel, err := svc.Create(1, CreateAIModelRequest{Name: "backup-summary", BaseURL: backup.URL})
	require.NoError(t, err)
	primaryModel, err := svc.Create(1, CreateAIModelRequest{
		Name:          "primary-summary",
		BaseURL:       primary.URL,
		BackupModelID: &backupModel.ID,
	})
	require.NoError(t, err)

	articles := []ArticleForSummary{{Title: "a", Content: "b", FeedTitle: "f", PublishedAt: "2025-03-01"}}
	got, err := svc.Summarize(1, primaryModel.ID, articles, nil)
	require.NoError(t, err)
	assert.Equal(t, "备用总结", got)
	assert.Equal(t, 1, backupCalls)
}

func TestAIModelService_SummarizeStream(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"中\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"文\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"总\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"结\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{
		Name:    "test-model",
		BaseURL: server.URL,
		APIKey:  "sk-test",
	})
	require.NoError(t, err)

	articles := []ArticleForSummary{{Title: "a", Content: "b", FeedTitle: "f", PublishedAt: "2025-03-01"}}
	var collected strings.Builder
	err = svc.SummarizeStream(1, m.ID, articles, nil, func(chunk string) error {
		collected.WriteString(chunk)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "中文总结", collected.String())
}

func TestAIModelService_SummarizeStream_UsesBackupModelWhenPrimaryFails(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer primary.Close()

	var backupCalls int
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupCalls++
		assert.Equal(t, "/chat/completions", r.URL.Path)
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "backup-stream", req.Model)
		assert.True(t, req.Stream)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"备\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"用\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer backup.Close()

	backupModel, err := svc.Create(1, CreateAIModelRequest{Name: "backup-stream", BaseURL: backup.URL})
	require.NoError(t, err)
	primaryModel, err := svc.Create(1, CreateAIModelRequest{
		Name:          "primary-stream",
		BaseURL:       primary.URL,
		BackupModelID: &backupModel.ID,
	})
	require.NoError(t, err)

	articles := []ArticleForSummary{{Title: "a", Content: "b", FeedTitle: "f", PublishedAt: "2025-03-01"}}
	var collected strings.Builder
	err = svc.SummarizeStream(1, primaryModel.ID, articles, nil, func(chunk string) error {
		collected.WriteString(chunk)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "备用", collected.String())
	assert.Equal(t, 1, backupCalls)
}
