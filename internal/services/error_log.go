package services

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

type ErrorLogService struct {
	db *gorm.DB
}

func NewErrorLogService(db *gorm.DB) *ErrorLogService {
	return &ErrorLogService{db: db}
}

type CreateErrorLogRequest struct {
	UserID   *uint
	Level    string
	Message  string
	Location string
	Method   string
	Path     string
	Status   int
	Stack    string
	Now      *time.Time
}

func callerLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	// 只保留末段路径，避免过长
	parts := strings.Split(file, "/")
	if len(parts) > 3 {
		file = strings.Join(parts[len(parts)-3:], "/")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func (s *ErrorLogService) Create(req CreateErrorLogRequest) error {
	if s == nil || s.db == nil {
		return nil
	}
	level := strings.ToLower(strings.TrimSpace(req.Level))
	if level == "" {
		level = "error"
	}
	now := time.Now()
	if req.Now != nil {
		now = *req.Now
	}
	loc := strings.TrimSpace(req.Location)
	if loc == "" {
		loc = callerLocation(3)
	}
	item := &models.ErrorLog{
		UserID:   req.UserID,
		Level:    level,
		Message:  strings.TrimSpace(req.Message),
		Location: loc,
		Method:   strings.TrimSpace(req.Method),
		Path:     strings.TrimSpace(req.Path),
		Status:   req.Status,
		Stack:    req.Stack,
		CreatedAt: now,
	}
	return s.db.Create(item).Error
}

type ListErrorLogsRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (s *ErrorLogService) List(userID uint, req ListErrorLogsRequest) ([]models.ErrorLog, int64, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	// 同时包含：属于该用户的日志 + user_id 为空的系统级日志
	q := s.db.Model(&models.ErrorLog{}).
		Where("(user_id = ? OR user_id IS NULL)", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.ErrorLog
	if err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ErrorLogService) Delete(userID uint, id uint) error {
	res := s.db.Where("(user_id = ? OR user_id IS NULL) AND id = ?", userID, id).Delete(&models.ErrorLog{})
	return res.Error
}

