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
	ErrSummaryScheduleNotFound = errors.New("定时总结配置不存在")
)

type SummaryScheduleService struct {
	db *gorm.DB
}

func NewSummaryScheduleService(db *gorm.DB) *SummaryScheduleService {
	return &SummaryScheduleService{db: db}
}

type CreateSummaryScheduleRequest struct {
	AIModelID uint   `json:"ai_model_id" binding:"required"`
	FeedIDs   []uint `json:"feed_ids"`
	RunAt     string `json:"run_at" binding:"required"` // HH:MM
	PageSize  int    `json:"page_size"`
	Order     string `json:"order"`
	Enabled   *bool  `json:"enabled"`
}

type UpdateSummaryScheduleRequest struct {
	AIModelID uint   `json:"ai_model_id" binding:"required"`
	FeedIDs   []uint `json:"feed_ids"`
	RunAt     string `json:"run_at" binding:"required"`
	PageSize  int    `json:"page_size"`
	Order     string `json:"order"`
	Enabled   *bool  `json:"enabled"`
}

func normalizeRunAt(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("run_at 不能为空")
	}
	// 仅校验 HH:MM 解析
	if _, err := time.Parse("15:04", s); err != nil {
		return "", errors.New("run_at 格式错误，请使用 HH:MM")
	}
	return s, nil
}

func (s *SummaryScheduleService) List(userID uint) ([]models.AISummarySchedule, error) {
	var items []models.AISummarySchedule
	err := s.db.Where("user_id = ?", userID).Order("id DESC").Find(&items).Error
	return items, err
}

func (s *SummaryScheduleService) Create(userID uint, req CreateSummaryScheduleRequest) (*models.AISummarySchedule, error) {
	runAt, err := normalizeRunAt(req.RunAt)
	if err != nil {
		return nil, err
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	b, err := json.Marshal(req.FeedIDs)
	if err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m := &models.AISummarySchedule{
		UserID:     userID,
		AIModelID:  req.AIModelID,
		FeedIDsJSON: string(b),
		RunAt:      runAt,
		PageSize:   pageSize,
		Order:      normalizeOrder(req.Order),
		Enabled:    enabled,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SummaryScheduleService) Update(userID uint, id uint, req UpdateSummaryScheduleRequest) (*models.AISummarySchedule, error) {
	var m models.AISummarySchedule
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSummaryScheduleNotFound
		}
		return nil, err
	}
	runAt, err := normalizeRunAt(req.RunAt)
	if err != nil {
		return nil, err
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	b, err := json.Marshal(req.FeedIDs)
	if err != nil {
		return nil, err
	}
	m.AIModelID = req.AIModelID
	m.FeedIDsJSON = string(b)
	m.RunAt = runAt
	m.PageSize = pageSize
	m.Order = normalizeOrder(req.Order)
	if req.Enabled != nil {
		m.Enabled = *req.Enabled
	}
	if err := s.db.Save(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SummaryScheduleService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.AISummarySchedule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSummaryScheduleNotFound
	}
	return nil
}

