package services

import (
	"encoding/json"
	"errors"
	"strings"

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

	AIEnabled *bool `json:"ai_enabled"`  // 可选：创建时配置
	AIModelID *uint `json:"ai_model_id"` // 可选：创建时配置，0=清空
	// AIClassifierPrompt 可选；nil 不写库；空字符串表示使用内置默认（存库为空）
	AIClassifierPrompt *string `json:"ai_classifier_prompt"`
}

// Create 添加订阅
func (s *FeedService) Create(userID uint, req CreateFeedRequest) (*FeedWithAI, error) {
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

	// 创建时可选写入 AI 设置
	if req.AIEnabled != nil || req.AIModelID != nil || req.AIClassifierPrompt != nil {
		if err := s.upsertFeedAISetting(userID, feed.ID, req.AIEnabled, req.AIModelID, req.AIClassifierPrompt); err != nil {
			return nil, err
		}
	}
	return s.getFeedWithAI(userID, feed.ID)
}

// List 获取用户订阅列表
func (s *FeedService) List(userID uint) ([]FeedWithAI, error) {
	var feeds []models.Feed
	if err := s.db.Preload("Category").Preload("Proxy").Where("user_id = ?", userID).Order("created_at DESC").Find(&feeds).Error; err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return []FeedWithAI{}, nil
	}
	feedIDs := make([]uint, 0, len(feeds))
	for i := range feeds {
		feedIDs = append(feedIDs, feeds[i].ID)
	}
	var settings []models.FeedAISetting
	_ = s.db.Where("user_id = ? AND feed_id IN ?", userID, feedIDs).Find(&settings).Error
	settingByFeedID := make(map[uint]models.FeedAISetting, len(settings))
	for _, st := range settings {
		settingByFeedID[st.FeedID] = st
	}
	out := make([]FeedWithAI, 0, len(feeds))
	for _, f := range feeds {
		st, ok := settingByFeedID[f.ID]
		if !ok {
			out = append(out, FeedWithAI{Feed: f})
			continue
		}
		out = append(out, s.decorateFeedWithAI(f, &st))
	}
	return out, nil
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

	AIEnabled *bool `json:"ai_enabled"` // nil=不修改
	AIModelID *uint `json:"ai_model_id"` // nil=不修改，0=清空
	// AIClassifierPrompt nil=不修改；指向空字符串=清空为默认（存库空）
	AIClassifierPrompt *string `json:"ai_classifier_prompt"`
}

type FeedWithAI struct {
	models.Feed
	AIEnabled          bool     `json:"ai_enabled"`
	AIModelID          *uint    `json:"ai_model_id"`
	AIClassifierPrompt string   `json:"ai_classifier_prompt"`
	AISummary          string   `json:"ai_summary"`
	AICategories       []string `json:"ai_categories"`
}

// Update 更新订阅设置
func (s *FeedService) Update(userID uint, id uint, req UpdateFeedRequest) (*FeedWithAI, error) {
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
	if req.AIEnabled != nil || req.AIModelID != nil || req.AIClassifierPrompt != nil {
		if err := s.upsertFeedAISetting(userID, feed.ID, req.AIEnabled, req.AIModelID, req.AIClassifierPrompt); err != nil {
			return nil, err
		}
	}
	return s.getFeedWithAI(userID, feed.ID)
}

func (s *FeedService) getFeedWithAI(userID uint, feedID uint) (*FeedWithAI, error) {
	var f models.Feed
	if err := s.db.Preload("Category").Preload("Proxy").Where("user_id = ? AND id = ?", userID, feedID).First(&f).Error; err != nil {
		return nil, err
	}
	var st models.FeedAISetting
	err := s.db.Where("user_id = ? AND feed_id = ?", userID, feedID).First(&st).Error
	if err != nil {
		return &FeedWithAI{Feed: f}, nil
	}
	out := s.decorateFeedWithAI(f, &st)
	return &out, nil
}

func (s *FeedService) decorateFeedWithAI(f models.Feed, st *models.FeedAISetting) FeedWithAI {
	var cats []string
	raw := strings.TrimSpace(st.CategoriesJSON)
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cats)
	}
	return FeedWithAI{
		Feed:               f,
		AIEnabled:          st.Enabled,
		AIModelID:          st.AIModelID,
		AIClassifierPrompt: st.ClassifierPrompt,
		AISummary:          strings.TrimSpace(st.Summary),
		AICategories:       cats,
	}
}

func (s *FeedService) upsertFeedAISetting(userID uint, feedID uint, enabled *bool, aiModelID *uint, classifierPrompt *string) error {
	var st models.FeedAISetting
	err := s.db.Where("user_id = ? AND feed_id = ?", userID, feedID).First(&st).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			st = models.FeedAISetting{UserID: userID, FeedID: feedID}
			if enabled != nil {
				st.Enabled = *enabled
			}
			if aiModelID != nil {
				if *aiModelID == 0 {
					st.AIModelID = nil
				} else {
					st.AIModelID = aiModelID
				}
			}
			if classifierPrompt != nil {
				st.ClassifierPrompt = *classifierPrompt
			}
			return s.db.Create(&st).Error
		}
		return err
	}
	updates := map[string]any{}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if aiModelID != nil {
		if *aiModelID == 0 {
			updates["ai_model_id"] = nil
		} else {
			updates["ai_model_id"] = *aiModelID
		}
	}
	if classifierPrompt != nil {
		updates["classifier_prompt"] = *classifierPrompt
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&models.FeedAISetting{}).Where("id = ?", st.ID).Updates(updates).Error
}

// Delete 删除订阅
func (s *FeedService) Delete(userID uint, id uint) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Feed{})
	if result.RowsAffected == 0 {
		return ErrFeedNotFound
	}
	return result.Error
}
