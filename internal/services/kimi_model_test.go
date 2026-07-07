package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
)

func TestNormalizeModelName(t *testing.T) {
	assert.Equal(t, "kimi-k2.6", normalizeModelName("  Kimi-K2.6  "))
}

func TestIsKimiModelDetection(t *testing.T) {
	assert.True(t, isKimiK26Model("kimi-k2.6"))
	assert.True(t, isKimiK26Model("KIMI-K2.6"))
	assert.False(t, isKimiK26Model("kimi-k2.5"))

	assert.True(t, isKimiK27CodeModel("kimi-k2.7-code"))
	assert.True(t, isKimiK27CodeModel("kimi-k2.7-code-highspeed"))
	assert.False(t, isKimiK27CodeModel("kimi-k2.6"))

	assert.True(t, isKimiFixedSamplingModel("kimi-k2.6"))
	assert.True(t, isKimiFixedSamplingModel("kimi-k2.7-code"))
	assert.False(t, isKimiFixedSamplingModel("gpt-4o"))
}

func TestBuildChatCompletionsRequest_KimiK26(t *testing.T) {
	m := &models.AIModel{Name: "kimi-k2.6"}
	req := buildChatCompletionsRequest(m, 2000, false, []chatMessage{{Role: "user", Content: "hi"}})
	require.NotNil(t, req.Thinking)
	assert.Equal(t, "disabled", req.Thinking.Type)
	require.NotNil(t, req.TopP)
	assert.Equal(t, defaultKimiTopP, *req.TopP)
	require.NotNil(t, req.N)
	assert.Equal(t, defaultKimiN, *req.N)
	require.NotNil(t, req.PresencePenalty)
	assert.Equal(t, defaultKimiPresencePenalty, *req.PresencePenalty)
	require.NotNil(t, req.FrequencyPenalty)
	assert.Equal(t, defaultKimiFrequencyPenalty, *req.FrequencyPenalty)

	raw, err := json.Marshal(req)
	require.NoError(t, err)
	body := string(raw)
	assert.Contains(t, body, `"thinking":{"type":"disabled"}`)
	assert.NotContains(t, body, "temperature")
	assert.Contains(t, body, `"top_p":0.95`)
	assert.Contains(t, body, `"n":1`)
	assert.Contains(t, body, `"presence_penalty":0`)
	assert.Contains(t, body, `"frequency_penalty":0`)
}

func TestBuildChatCompletionsRequest_KimiK26_CustomSampling(t *testing.T) {
	topP := 0.8
	n := 2
	presence := 0.1
	frequency := -0.2
	m := &models.AIModel{
		Name:             "kimi-k2.6",
		TopP:             &topP,
		N:                &n,
		PresencePenalty:  &presence,
		FrequencyPenalty: &frequency,
	}
	req := buildChatCompletionsRequest(m, 2000, false, nil)
	require.NotNil(t, req.TopP)
	assert.Equal(t, 0.8, *req.TopP)
	require.NotNil(t, req.N)
	assert.Equal(t, 2, *req.N)
	require.NotNil(t, req.PresencePenalty)
	assert.Equal(t, 0.1, *req.PresencePenalty)
	require.NotNil(t, req.FrequencyPenalty)
	assert.Equal(t, -0.2, *req.FrequencyPenalty)
}

func TestBuildChatCompletionsRequest_KimiK27Code(t *testing.T) {
	m := &models.AIModel{Name: "kimi-k2.7-code"}
	req := buildChatCompletionsRequest(m, 4096, true, []chatMessage{{Role: "user", Content: "code"}})
	assert.Nil(t, req.Thinking)
	assert.True(t, req.Stream)
	require.NotNil(t, req.TopP)
	assert.Equal(t, defaultKimiTopP, *req.TopP)

	raw, err := json.Marshal(req)
	require.NoError(t, err)
	body := string(raw)
	assert.NotContains(t, body, "thinking")
	assert.NotContains(t, body, "temperature")
	assert.Contains(t, body, `"top_p":0.95`)
	assert.Contains(t, body, `"n":1`)
	assert.Contains(t, body, `"presence_penalty":0`)
	assert.Contains(t, body, `"frequency_penalty":0`)
}

func TestBuildChatCompletionsRequest_GenericModel(t *testing.T) {
	m := &models.AIModel{Name: "gpt-4o-mini"}
	req := buildChatCompletionsRequest(m, 100, false, nil)
	assert.Nil(t, req.Thinking)
	assert.Nil(t, req.TopP)
	assert.Nil(t, req.N)
	assert.Nil(t, req.PresencePenalty)
	assert.Nil(t, req.FrequencyPenalty)
}

func TestBuildChatCompletionsRequest_KimiK27CodeHighspeed(t *testing.T) {
	m := &models.AIModel{Name: "kimi-k2.7-code-highspeed"}
	req := buildChatCompletionsRequest(m, 4096, false, nil)
	require.NotNil(t, req.TopP)
	assert.Equal(t, defaultKimiTopP, *req.TopP)
	assert.Nil(t, req.Thinking)
}

func TestAIModelService_ChatCompletionText_KimiK26_SendsDisabledThinking(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		requestBodyBytes, _ := json.Marshal(req)
		requestBody = string(requestBodyBytes)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{Name: "kimi-k2.6", BaseURL: server.URL})
	require.NoError(t, err)

	got, err := svc.ChatCompletionText(1, m.ID, 2000, []chatMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "ok", got)
	assert.Contains(t, requestBody, `"thinking":{"type":"disabled"}`)
	assert.Contains(t, requestBody, `"top_p":0.95`)
	assert.Contains(t, requestBody, `"n":1`)
	assert.NotContains(t, requestBody, "temperature")
}

func TestAIModelService_ChatCompletionText_KimiK26_UsesStoredSamplingParams(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		requestBodyBytes, _ := json.Marshal(req)
		requestBody = string(requestBodyBytes)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	topP := 0.7
	n := 3
	presence := 0.5
	frequency := -0.5
	m, err := svc.Create(1, CreateAIModelRequest{
		Name:             "kimi-k2.6",
		BaseURL:          server.URL,
		TopP:             &topP,
		N:                &n,
		PresencePenalty:  &presence,
		FrequencyPenalty: &frequency,
	})
	require.NoError(t, err)

	_, err = svc.ChatCompletionText(1, m.ID, 2000, []chatMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Contains(t, requestBody, `"top_p":0.7`)
	assert.Contains(t, requestBody, `"n":3`)
	assert.Contains(t, requestBody, `"presence_penalty":0.5`)
	assert.Contains(t, requestBody, `"frequency_penalty":-0.5`)
}

func TestAIModelService_ChatCompletionText_KimiK27Code_OmitsThinking(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatCompletionsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		requestBodyBytes, _ := json.Marshal(req)
		requestBody = string(requestBodyBytes)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"code ok"}}]}`))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{Name: "kimi-k2.7-code", BaseURL: server.URL})
	require.NoError(t, err)

	got, err := svc.ChatCompletionText(1, m.ID, 4096, []chatMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "code ok", got)
	assert.NotContains(t, requestBody, "thinking")
	assert.Contains(t, requestBody, `"top_p":0.95`)
	assert.Contains(t, requestBody, `"n":1`)
	assert.NotContains(t, requestBody, "temperature")
}

func TestAIModelService_SummarizeStream_KimiK27Code_SkipsReasoningContent(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(`data: {"choices":[{"delta":{"reasoning_content":"思考中"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"最"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"终"}}]}` + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	m, err := svc.Create(1, CreateAIModelRequest{Name: "kimi-k2.7-code", BaseURL: server.URL})
	require.NoError(t, err)

	articles := []ArticleForSummary{{Title: "a", Content: "b", FeedTitle: "f", PublishedAt: "2025-03-01"}}
	var collected strings.Builder
	err = svc.SummarizeStream(1, m.ID, articles, nil, func(chunk string) error {
		collected.WriteString(chunk)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "最终", collected.String())
}

func TestAIModelService_Create_RejectsInvalidSamplingParams(t *testing.T) {
	db := setupAIModelDB(t)
	svc := NewAIModelService(db)

	invalidTopP := 1.5
	_, err := svc.Create(1, CreateAIModelRequest{
		Name:    "kimi-k2.6",
		BaseURL: "https://api.example/v1",
		TopP:    &invalidTopP,
	})
	assert.Error(t, err)
}
