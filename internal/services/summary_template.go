package services

import (
	"errors"
	"strings"

	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

var ErrSummaryTemplateNotFound = errors.New("总结模版不存在")

// PromptOptionsFromTemplate 将数据库模版转为总结 Prompt；nil 模版返回 nil（使用内置默认）
func PromptOptionsFromTemplate(t *models.AISummaryTemplate) *SummaryPromptOptions {
	if t == nil {
		return nil
	}
	return &SummaryPromptOptions{
		SystemPrompt: t.SystemPrompt,
		UserPrefix:   t.UserPromptPrefix,
	}
}

type SummaryTemplateService struct {
	db *gorm.DB
}

func NewSummaryTemplateService(db *gorm.DB) *SummaryTemplateService {
	return &SummaryTemplateService{db: db}
}

type CreateSummaryTemplateRequest struct {
	Name             string `json:"name" binding:"required"`
	SystemPrompt     string `json:"system_prompt"`
	UserPromptPrefix string `json:"user_prompt_prefix"`
	SortOrder        int    `json:"sort_order"`
}

type UpdateSummaryTemplateRequest struct {
	Name             string `json:"name" binding:"required"`
	SystemPrompt     string `json:"system_prompt"`
	UserPromptPrefix string `json:"user_prompt_prefix"`
	SortOrder        int    `json:"sort_order"`
}

func (s *SummaryTemplateService) List(userID uint) ([]models.AISummaryTemplate, error) {
	var items []models.AISummaryTemplate
	err := s.db.Where("user_id = ?", userID).Order("sort_order ASC, id DESC").Find(&items).Error
	return items, err
}

func (s *SummaryTemplateService) GetByID(userID uint, id uint) (*models.AISummaryTemplate, error) {
	var m models.AISummaryTemplate
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSummaryTemplateNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *SummaryTemplateService) Create(userID uint, req CreateSummaryTemplateRequest) (*models.AISummaryTemplate, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("模版名称不能为空")
	}
	if len([]rune(name)) > 128 {
		return nil, errors.New("模版名称过长")
	}
	m := &models.AISummaryTemplate{
		UserID:           userID,
		Name:             name,
		SystemPrompt:     req.SystemPrompt,
		UserPromptPrefix: req.UserPromptPrefix,
		SortOrder:        req.SortOrder,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SummaryTemplateService) Update(userID uint, id uint, req UpdateSummaryTemplateRequest) (*models.AISummaryTemplate, error) {
	m, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("模版名称不能为空")
	}
	if len([]rune(name)) > 128 {
		return nil, errors.New("模版名称过长")
	}
	m.Name = name
	m.SystemPrompt = req.SystemPrompt
	m.UserPromptPrefix = req.UserPromptPrefix
	m.SortOrder = req.SortOrder
	if err := s.db.Save(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SummaryTemplateService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.AISummaryTemplate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSummaryTemplateNotFound
	}
	return nil
}
