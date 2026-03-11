package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrSummaryHistoryNotFound = errors.New("总结历史不存在")
)

type SummaryHistoryService struct {
	db *gorm.DB
}

func NewSummaryHistoryService(db *gorm.DB) *SummaryHistoryService {
	return &SummaryHistoryService{db: db}
}

type CreateSummaryHistoryRequest struct {
	AIModelID    uint
	FeedIDs      []uint
	StartTime    string
	EndTime      string
	Page         int
	PageSize     int
	Order        string
	ArticleCount int
	Total        int64
	Content      string
	Error        string
	CreatedAt    *time.Time
}

func normalizeOrder(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "asc" {
		return "asc"
	}
	return "desc"
}

func (s *SummaryHistoryService) Create(userID uint, req CreateSummaryHistoryRequest) (*models.AISummaryHistory, error) {
	b, err := json.Marshal(req.FeedIDs)
	if err != nil {
		return nil, err
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	h := &models.AISummaryHistory{
		UserID:       userID,
		AIModelID:    req.AIModelID,
		FeedIDsJSON:  string(b),
		StartTime:    strings.TrimSpace(req.StartTime),
		EndTime:      strings.TrimSpace(req.EndTime),
		Page:         page,
		PageSize:     pageSize,
		Order:        normalizeOrder(req.Order),
		ArticleCount: req.ArticleCount,
		Total:        req.Total,
		Content:      req.Content,
		Error:        req.Error,
	}
	if req.CreatedAt != nil {
		h.CreatedAt = *req.CreatedAt
	}
	if err := s.db.Create(h).Error; err != nil {
		return nil, err
	}
	return h, nil
}

type ListSummaryHistoriesRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type SummaryHistoryItem struct {
	ID          uint      `json:"id"`
	AIModelID   uint      `json:"ai_model_id"`
	AIModelName string    `json:"ai_model_name"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
	Order       string    `json:"order"`
	ArticleCount int      `json:"article_count"`
	Total       int64     `json:"total"`
	Content     string    `json:"content"`
	Error       string    `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *SummaryHistoryService) List(userID uint, req ListSummaryHistoriesRequest) ([]SummaryHistoryItem, int64, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := s.db.Model(&models.AISummaryHistory{}).
		Where("ai_summary_histories.user_id = ?", userID).
		Joins("LEFT JOIN ai_models ON ai_models.id = ai_summary_histories.ai_model_id AND ai_models.deleted_at IS NULL")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		models.AISummaryHistory
		AIModelName string `gorm:"column:ai_model_name"`
	}
	var rows []row
	if err := q.Select("ai_summary_histories.*, ai_models.name AS ai_model_name").
		Order("ai_summary_histories.created_at DESC, ai_summary_histories.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]SummaryHistoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, SummaryHistoryItem{
			ID:           r.ID,
			AIModelID:    r.AIModelID,
			AIModelName:  r.AIModelName,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			Page:         r.Page,
			PageSize:     r.PageSize,
			Order:        r.Order,
			ArticleCount: r.ArticleCount,
			Total:        r.Total,
			Content:      r.Content,
			Error:        r.Error,
			CreatedAt:    r.CreatedAt,
		})
	}
	return items, total, nil
}

func (s *SummaryHistoryService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.AISummaryHistory{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSummaryHistoryNotFound
	}
	return nil
}

