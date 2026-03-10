package services

import (
	"errors"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrFeedNotFound = errors.New("订阅不存在")
)

// FeedService 订阅服务
type FeedService struct {
	db   *gorm.DB
	rss  *RSSService
}

// NewFeedService 创建订阅服务
func NewFeedService(db *gorm.DB, rss *RSSService) *FeedService {
	return &FeedService{db: db, rss: rss}
}

// CreateFeedRequest 创建订阅请求
type CreateFeedRequest struct {
	URL                   string `json:"url" binding:"required,url"`
	CategoryID            uint   `json:"category_id" binding:"required"`
	UpdateIntervalMinutes int    `json:"update_interval_minutes" binding:"required,min=5,max=10080"`
	ProxyID               *uint  `json:"proxy_id"`
	ExpireDays            *int   `json:"expire_days"` // nil=默认90天，0=永不过期，>0=保留天数
}

// Create 添加订阅
func (s *FeedService) Create(userID uint, req CreateFeedRequest) (*models.Feed, error) {
	// 校验分类归属与存在（放在抓取 RSS 之前，避免无意义的网络请求）
	var cat models.FeedCategory
	if err := s.db.Where("user_id = ? AND id = ?", userID, req.CategoryID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("分类不存在")
		}
		return nil, err
	}

	proxyURL := ""
	if req.ProxyID != nil && *req.ProxyID > 0 {
		var p models.Proxy
		if err := s.db.Where("user_id = ? AND id = ?", userID, *req.ProxyID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("代理不存在")
			}
			return nil, err
		}
		proxyURL = p.URL
	}

	title, err := s.rss.FetchAndParse(req.URL, proxyURL)
	if err != nil {
		return nil, err
	}
	var count int64
	s.db.Model(&models.Feed{}).Where("user_id = ? AND url = ?", userID, req.URL).Count(&count)
	if count > 0 {
		return nil, errors.New("订阅已存在")
	}
	expireDays := 90
	if req.ExpireDays != nil {
		if *req.ExpireDays >= 0 {
			expireDays = *req.ExpireDays
		}
	}
	feed := &models.Feed{
		UserID:                userID,
		CategoryID:            &req.CategoryID,
		ProxyID:               req.ProxyID,
		URL:                   req.URL,
		Title:                 title,
		UpdateIntervalMinutes: req.UpdateIntervalMinutes,
		ExpireDays:            expireDays,
	}
	if err := s.db.Create(feed).Error; err != nil {
		return nil, err
	}
	if expireDays == 0 {
		_ = s.db.Model(feed).Updates(map[string]interface{}{"expire_days": 0})
	}
	s.db.Preload("Proxy").First(feed, feed.ID)
	_ = s.rss.FetchFeed(feed)
	return feed, nil
}

// List 获取用户订阅列表
func (s *FeedService) List(userID uint) ([]models.Feed, error) {
	var feeds []models.Feed
	err := s.db.Preload("Category").Preload("Proxy").Where("user_id = ?", userID).Order("created_at DESC").Find(&feeds).Error
	return feeds, err
}

// GetByID 根据 ID 获取订阅
func (s *FeedService) GetByID(userID uint, id uint) (*models.Feed, error) {
	var feed models.Feed
	if err := s.db.Preload("Category").Preload("Proxy").Where("user_id = ? AND id = ?", userID, id).First(&feed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFeedNotFound
		}
		return nil, err
	}
	return &feed, nil
}

// UpdateFeedRequest 更新订阅请求
type UpdateFeedRequest struct {
	CategoryID            *uint `json:"category_id"`             // nil 表示不修改
	UpdateIntervalMinutes int   `json:"update_interval_minutes" binding:"required,min=5,max=10080"`
	ProxyID               *uint `json:"proxy_id"`
	ExpireDays            *int  `json:"expire_days"` // 0=永不过期，nil 表示不修改
}

// Update 更新订阅设置
func (s *FeedService) Update(userID uint, id uint, req UpdateFeedRequest) (*models.Feed, error) {
	feed, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	if req.CategoryID != nil {
		if *req.CategoryID > 0 {
			var cat models.FeedCategory
			if err := s.db.Where("user_id = ? AND id = ?", userID, *req.CategoryID).First(&cat).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.New("分类不存在")
				}
				return nil, err
			}
		}
	}
	if req.ProxyID != nil && *req.ProxyID > 0 {
		var p models.Proxy
		if err := s.db.Where("user_id = ? AND id = ?", userID, *req.ProxyID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("代理不存在")
			}
			return nil, err
		}
	}
	updates := map[string]interface{}{"update_interval_minutes": req.UpdateIntervalMinutes}
	if req.ProxyID != nil {
		updates["proxy_id"] = *req.ProxyID
	}
	if req.ExpireDays != nil {
		updates["expire_days"] = *req.ExpireDays
	}
	if err := s.db.Model(feed).Updates(updates).Error; err != nil {
		return nil, err
	}
	if req.ProxyID == nil {
		_ = s.db.Model(feed).Update("proxy_id", nil)
	}
	if req.CategoryID != nil {
		var catVal interface{} = nil
		if *req.CategoryID > 0 {
			catVal = *req.CategoryID
		}
		if err := s.db.Table("feeds").Where("id = ? AND user_id = ?", feed.ID, userID).Update("category_id", catVal).Error; err != nil {
			return nil, err
		}
	}
	var result models.Feed
	if err := s.db.Preload("Category").Preload("Proxy").First(&result, feed.ID).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete 删除订阅
func (s *FeedService) Delete(userID uint, id uint) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Feed{})
	if result.RowsAffected == 0 {
		return ErrFeedNotFound
	}
	return result.Error
}
