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
	err := s.db.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&items).Error
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
	var maxOrder int
	s.db.Model(&models.FeedCategory{}).Where("user_id = ?", userID).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxOrder)
	cat := &models.FeedCategory{
		UserID:    userID,
		Name:      name,
		SortOrder: maxOrder + 1,
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

// Reorder 按 id_list 顺序更新 sort_order（id_list 为当前用户下的分类 id 有序列表）
func (s *CategoryService) Reorder(userID uint, idList []uint) error {
	for i, id := range idList {
		res := s.db.Model(&models.FeedCategory{}).Where("user_id = ? AND id = ?", userID, id).Update("sort_order", i)
		if res.Error != nil {
			return res.Error
		}
	}
	return nil
}

