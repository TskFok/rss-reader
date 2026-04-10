package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrAIModelNotFound = errors.New("AI 模型不存在")
)

type AIModelService struct {
	db *gorm.DB
}

func NewAIModelService(db *gorm.DB) *AIModelService {
	return &AIModelService{db: db}
}

type CreateAIModelRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=128"`
	BaseURL       string `json:"base_url" binding:"required,min=1,max=512"`
	APIKey        string `json:"api_key"`
	BackupModelID *uint  `json:"backup_model_id"`
}

type UpdateAIModelRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=128"`
	BaseURL       string  `json:"base_url" binding:"required,min=1,max=512"`
	APIKey        *string `json:"api_key"` // nil 表示不修改，空字符串表示清空
	BackupModelID *uint   `json:"backup_model_id"`
}

func normalizeURL(s string) string {
	return strings.TrimSpace(s)
}

func (s *AIModelService) List(userID uint) ([]models.AIModel, error) {
	var items []models.AIModel
	err := s.db.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

func (s *AIModelService) Create(userID uint, req CreateAIModelRequest) (*models.AIModel, error) {
	baseURL := normalizeURL(req.BaseURL)
	if baseURL == "" {
		return nil, errors.New("调用地址不能为空")
	}
	var backupID *uint
	if req.BackupModelID != nil && *req.BackupModelID != 0 {
		var backup models.AIModel
		if err := s.db.Where("user_id = ? AND id = ?", userID, *req.BackupModelID).First(&backup).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("备用模型不存在或不属于当前用户")
			}
			return nil, err
		}
		backupID = &backup.ID
	}
	var maxOrder int
	s.db.Model(&models.AIModel{}).Where("user_id = ?", userID).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxOrder)
	m := &models.AIModel{
		UserID:        userID,
		Name:          strings.TrimSpace(req.Name),
		BaseURL:       baseURL,
		APIKey:        strings.TrimSpace(req.APIKey),
		BackupModelID: backupID,
		SortOrder:     maxOrder + 1,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *AIModelService) GetByID(userID uint, id uint) (*models.AIModel, error) {
	var m models.AIModel
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIModelNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *AIModelService) Update(userID uint, id uint, req UpdateAIModelRequest) (*models.AIModel, error) {
	m, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	if req.BackupModelID != nil {
		if *req.BackupModelID == 0 {
			m.BackupModelID = nil
		} else if *req.BackupModelID == id {
			return nil, errors.New("备用模型不能是自己")
		} else {
			var backup models.AIModel
			if err := s.db.Where("user_id = ? AND id = ?", userID, *req.BackupModelID).First(&backup).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.New("备用模型不存在或不属于当前用户")
				}
				return nil, err
			}
			m.BackupModelID = &backup.ID
		}
	}
	baseURL := normalizeURL(req.BaseURL)
	if baseURL == "" {
		return nil, errors.New("调用地址不能为空")
	}
	m.Name = strings.TrimSpace(req.Name)
	m.BaseURL = baseURL
	if req.APIKey != nil {
		m.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if err := s.db.Save(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *AIModelService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.AIModel{})
	if res.RowsAffected == 0 {
		return ErrAIModelNotFound
	}
	return res.Error
}

// Reorder 按 id_list 顺序更新 sort_order（id_list 为当前用户下的模型 id 有序列表）
func (s *AIModelService) Reorder(userID uint, idList []uint) error {
	for i, id := range idList {
		res := s.db.Model(&models.AIModel{}).Where("user_id = ? AND id = ?", userID, id).Update("sort_order", i)
		if res.Error != nil {
			return res.Error
		}
	}
	return nil
}

// chatCompletionsRequest OpenAI 兼容的聊天请求
type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionText 非流式调用 OpenAI 兼容 chat/completions，返回首条回复正文
func (s *AIModelService) ChatCompletionText(userID uint, modelID uint, maxTokens int, messages []chatMessage) (string, error) {
	m, err := s.GetByID(userID, modelID)
	if err != nil {
		return "", err
	}
	baseURL := strings.TrimSuffix(m.BaseURL, "/")
	chatURL := baseURL
	if !strings.HasSuffix(chatURL, "/chat/completions") {
		chatURL = baseURL + "/chat/completions"
	}
	body := chatCompletionsRequest{
		Model:     m.Name,
		MaxTokens: maxTokens,
		Stream:    false,
		Messages:  messages,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.APIKey)
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("模型调用失败: HTTP " + resp.Status)
	}
	var respBody chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", err
	}
	if len(respBody.Choices) == 0 || respBody.Choices[0].Message.Content == "" {
		return "", errors.New("模型未返回有效内容")
	}
	return strings.TrimSpace(respBody.Choices[0].Message.Content), nil
}

// chatCompletionsResponse OpenAI 兼容的聊天响应
type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Summarize 使用指定 AI 模型对文章列表生成总结；opts 为 nil 时使用内置中文说明前缀
func (s *AIModelService) Summarize(userID uint, modelID uint, articles []ArticleForSummary, opts *SummaryPromptOptions) (string, error) {
	if len(articles) == 0 {
		return "", errors.New("没有可总结的文章")
	}
	m, err := s.GetByID(userID, modelID)
	if err != nil {
		return "", err
	}
	msgs := BuildSummaryChatMessages(opts, articles)
	baseURL := strings.TrimSuffix(m.BaseURL, "/")
	chatURL := baseURL
	if !strings.HasSuffix(chatURL, "/chat/completions") {
		chatURL = baseURL + "/chat/completions"
	}
	body := chatCompletionsRequest{
		Model:     m.Name,
		MaxTokens: 2000,
		Messages:  msgs,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.APIKey)
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("模型调用失败: HTTP " + resp.Status)
	}
	var respBody chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", err
	}
	if len(respBody.Choices) == 0 || respBody.Choices[0].Message.Content == "" {
		return "", errors.New("模型未返回有效内容")
	}
	return strings.TrimSpace(respBody.Choices[0].Message.Content), nil
}

// streamChunk OpenAI 流式响应中的单个 chunk
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// SummarizeStream 流式生成 AI 总结；opts 为 nil 时使用内置说明前缀
func (s *AIModelService) SummarizeStream(userID uint, modelID uint, articles []ArticleForSummary, opts *SummaryPromptOptions, onChunk func(string) error) error {
	if len(articles) == 0 {
		return errors.New("没有可总结的文章")
	}
	m, err := s.GetByID(userID, modelID)
	if err != nil {
		return err
	}
	msgs := BuildSummaryChatMessages(opts, articles)
	baseURL := strings.TrimSuffix(m.BaseURL, "/")
	chatURL := baseURL
	if !strings.HasSuffix(chatURL, "/chat/completions") {
		chatURL = baseURL + "/chat/completions"
	}
	body := chatCompletionsRequest{
		Model:     m.Name,
		MaxTokens: 2000,
		Stream:    true,
		Messages:  msgs,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.APIKey)
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("模型调用失败: HTTP " + resp.Status)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if err := onChunk(chunk.Choices[0].Delta.Content); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

// Test 检测模型是否可用，向 API 发送简单请求
func (s *AIModelService) Test(userID uint, id uint) error {
	m, err := s.GetByID(userID, id)
	if err != nil {
		return err
	}
	baseURL := strings.TrimSuffix(m.BaseURL, "/")
	chatURL := baseURL
	if !strings.HasSuffix(chatURL, "/chat/completions") {
		chatURL = baseURL + "/chat/completions"
	}
	body := chatCompletionsRequest{
		Model:     m.Name,
		MaxTokens: 5,
		Messages:  []chatMessage{{Role: "user", Content: "hi"}},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.APIKey)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return errors.New("模型调用失败: HTTP " + resp.Status)
}
