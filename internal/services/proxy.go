package services

import (
	"errors"
	"strings"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrProxyNotFound = errors.New("代理不存在")
)

type ProxyService struct {
	db *gorm.DB
}

func NewProxyService(db *gorm.DB) *ProxyService {
	return &ProxyService{db: db}
}

type CreateProxyRequest struct {
	Name string `json:"name"`
	URL  string `json:"url" binding:"required,min=1,max=512"`
}

type UpdateProxyRequest struct {
	Name string `json:"name"`
	URL  string `json:"url" binding:"required,min=1,max=512"`
}

func normalizeProxyURL(s string) string {
	return strings.TrimSpace(s)
}

func (s *ProxyService) List(userID uint) ([]models.Proxy, error) {
	var items []models.Proxy
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *ProxyService) Create(userID uint, req CreateProxyRequest) (*models.Proxy, error) {
	url := normalizeProxyURL(req.URL)
	if url == "" {
		return nil, errors.New("代理地址不能为空")
	}
	proxy := &models.Proxy{
		UserID: userID,
		Name:   strings.TrimSpace(req.Name),
		URL:    url,
	}
	if err := s.db.Create(proxy).Error; err != nil {
		return nil, err
	}
	return proxy, nil
}

func (s *ProxyService) GetByID(userID uint, id uint) (*models.Proxy, error) {
	var proxy models.Proxy
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProxyNotFound
		}
		return nil, err
	}
	return &proxy, nil
}

func (s *ProxyService) Update(userID uint, id uint, req UpdateProxyRequest) (*models.Proxy, error) {
	proxy, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	url := normalizeProxyURL(req.URL)
	if url == "" {
		return nil, errors.New("代理地址不能为空")
	}
	proxy.Name = strings.TrimSpace(req.Name)
	proxy.URL = url
	if err := s.db.Save(proxy).Error; err != nil {
		return nil, err
	}
	return proxy, nil
}

func (s *ProxyService) Delete(userID uint, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Proxy{})
	if res.RowsAffected == 0 {
		return ErrProxyNotFound
	}
	return res.Error
}
