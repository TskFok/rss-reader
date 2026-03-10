package services

import (
	"errors"
	"strings"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrCategoryNameExists = errors.New("分类名称已存在")
)

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}

type UpdateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}

func normalizeCategoryName(s string) string {
	return strings.TrimSpace(s)
}

func (s *CategoryService) List(userID uint) ([]models.FeedCategory, error) {
	var items []models.FeedCategory
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *CategoryService) Create(userID uint, req CreateCategoryRequest) (*models.FeedCategory, error) {
	name := normalizeCategoryName(req.Name)
	if name == "" {
		return nil, errors.New("分类名称不能为空")
	}
	var exists int64
	s.db.Model(&models.FeedCategory{}).Where("user_id = ? AND name = ?", userID, name).Count(&exists)
	if exists > 0 {
		return nil, ErrCategoryNameExists
	}
	cat := &models.FeedCategory{
		UserID: userID,
		Name:   name,
	}
	if err := s.db.Create(cat).Error; err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *CategoryService) GetByID(userID uint, id uint) (*models.FeedCategory, error) {
	var cat models.FeedCategory
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &cat, nil
}

func (s *CategoryService) Update(userID uint, id uint, req UpdateCategoryRequest) (*models.FeedCategory, error) {
	cat, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	name := normalizeCategoryName(req.Name)
	if name == "" {
		return nil, errors.New("分类名称不能为空")
	}
	var exists int64
	s.db.Model(&models.FeedCategory{}).
		Where("user_id = ? AND name = ? AND id <> ?", userID, name, id).
		Count(&exists)
	if exists > 0 {
		return nil, ErrCategoryNameExists
	}
	if err := s.db.Model(cat).Update("name", name).Error; err != nil {
		return nil, err
	}
	cat.Name = name
	return cat, nil
}

func (s *CategoryService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.FeedCategory{})
	if res.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return res.Error
}

