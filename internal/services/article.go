package services

import (
	"errors"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var (
	ErrArticleNotFound = errors.New("文章不存在")
)

// ArticleService 文章服务
type ArticleService struct {
	db *gorm.DB
}

// NewArticleService 创建文章服务
func NewArticleService(db *gorm.DB) *ArticleService {
	return &ArticleService{db: db}
}

// ListArticlesRequest 文章列表请求
type ListArticlesRequest struct {
	FeedID   *uint  `form:"feed_id"`
	Read     *bool  `form:"read"`
	Favorite *bool  `form:"favorite"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ArticleWithRead 带阅读状态与收藏的文章
type ArticleWithRead struct {
	models.Article
	Read      bool   `json:"read"`
	Favorite  bool   `json:"favorite"`
	FeedTitle string `json:"feed_title"`
}

// List 获取用户可见的文章列表（通过 feed 归属）
func (s *ArticleService) List(userID uint, req ListArticlesRequest) ([]ArticleWithRead, int64, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := s.db.Model(&models.Article{}).
		Joins("JOIN feeds ON feeds.id = articles.feed_id AND feeds.deleted_at IS NULL").
		Where("feeds.user_id = ?", userID)
	if req.FeedID != nil {
		q = q.Where("articles.feed_id = ?", *req.FeedID)
	}
	if req.Read != nil {
		if *req.Read {
			q = q.Joins("JOIN user_articles ua_read ON ua_read.article_id = articles.id AND ua_read.user_id = ? AND ua_read.read_status = 1", userID)
		} else {
			q = q.Joins("LEFT JOIN user_articles ua_read ON ua_read.article_id = articles.id AND ua_read.user_id = ?", userID).
				Where("ua_read.id IS NULL OR ua_read.read_status = 0")
		}
	}
	if req.Favorite != nil && *req.Favorite {
		q = q.Joins("JOIN user_articles ua_fav ON ua_fav.article_id = articles.id AND ua_fav.user_id = ? AND ua_fav.favorite = 1", userID)
	}
	// 未筛选读/未读时，需按未读优先排序，需 LEFT JOIN 获取阅读状态
	if req.Read == nil {
		q = q.Joins("LEFT JOIN user_articles ua_sort ON ua_sort.article_id = articles.id AND ua_sort.user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var articles []models.Article
	offset := (page - 1) * pageSize
	orderClause := "articles.published_at DESC, articles.created_at DESC"
	if req.Read == nil {
		orderClause = "COALESCE(ua_sort.read_status, 0) ASC, " + orderClause
	}
	if err := q.Order(orderClause).
		Offset(offset).Limit(pageSize).
		Preload("Feed").
		Find(&articles).Error; err != nil {
		return nil, 0, err
	}
	if len(articles) == 0 {
		return []ArticleWithRead{}, total, nil
	}
	ids := make([]uint, len(articles))
	for i := range articles {
		ids[i] = articles[i].ID
	}
	var uas []models.UserArticle
	s.db.Where("user_id = ? AND article_id IN ?", userID, ids).Find(&uas)
	readMap := make(map[uint]bool)
	favMap := make(map[uint]bool)
	for _, ua := range uas {
		if ua.ReadStatus {
			readMap[ua.ArticleID] = true
		}
		if ua.Favorite {
			favMap[ua.ArticleID] = true
		}
	}
	result := make([]ArticleWithRead, len(articles))
	for i, a := range articles {
		feedTitle := ""
		if a.Feed.ID != 0 {
			feedTitle = a.Feed.Title
		}
		result[i] = ArticleWithRead{Article: a, Read: readMap[a.ID], Favorite: favMap[a.ID], FeedTitle: feedTitle}
	}
	return result, total, nil
}

// MarkRead 标记文章已读
func (s *ArticleService) MarkRead(userID uint, articleID uint) error {
	var article models.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	var feed models.Feed
	if err := s.db.First(&feed, article.FeedID).Error; err != nil {
		return ErrArticleNotFound
	}
	if feed.UserID != userID {
		return ErrArticleNotFound
	}
	var ua models.UserArticle
	err := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&ua).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ua = models.UserArticle{UserID: userID, ArticleID: articleID, ReadStatus: true}
			return s.db.Create(&ua).Error
		}
		return err
	}
	return s.db.Model(&ua).Update("read_status", true).Error
}

// ToggleFavorite 切换文章收藏状态
func (s *ArticleService) ToggleFavorite(userID uint, articleID uint) (bool, error) {
	var article models.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrArticleNotFound
		}
		return false, err
	}
	var feed models.Feed
	if err := s.db.First(&feed, article.FeedID).Error; err != nil {
		return false, ErrArticleNotFound
	}
	if feed.UserID != userID {
		return false, ErrArticleNotFound
	}
	var ua models.UserArticle
	err := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&ua).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ua = models.UserArticle{UserID: userID, ArticleID: articleID, Favorite: true}
			return true, s.db.Create(&ua).Error
		}
		return false, err
	}
	next := !ua.Favorite
	return next, s.db.Model(&ua).Update("favorite", next).Error
}

// CleanupExpiredArticles 删除各订阅下过期的文章（按 feed.expire_days 计算，0=永不过期），收藏的文章不删除
func (s *ArticleService) CleanupExpiredArticles() (int64, error) {
	var feeds []models.Feed
	if err := s.db.Where("expire_days > ?", 0).Select("id", "expire_days").Find(&feeds).Error; err != nil {
		return 0, err
	}
	var totalDeleted int64
	for _, f := range feeds {
		cutoff := time.Now().AddDate(0, 0, -f.ExpireDays)
		var ids []uint
		// 使用 published_at，若无则用 created_at；超过 expire_days 天的文章视为过期
		if err := s.db.Model(&models.Article{}).Where("feed_id = ?", f.ID).
			Where("COALESCE(published_at, created_at) < ?", cutoff).
			Pluck("id", &ids).Error; err != nil {
			continue
		}
		if len(ids) == 0 {
			continue
		}
		var favorited []uint
		s.db.Model(&models.UserArticle{}).Where("article_id IN ? AND favorite = 1", ids).Pluck("article_id", &favorited)
		favSet := make(map[uint]bool)
		for _, id := range favorited {
			favSet[id] = true
		}
		var toDelete []uint
		for _, id := range ids {
			if !favSet[id] {
				toDelete = append(toDelete, id)
			}
		}
		if len(toDelete) == 0 {
			continue
		}
		if res := s.db.Unscoped().Where("article_id IN ?", toDelete).Delete(&models.UserArticle{}); res.Error == nil {
			totalDeleted += res.RowsAffected
		}
		if res := s.db.Unscoped().Where("id IN ?", toDelete).Delete(&models.Article{}); res.Error == nil {
			totalDeleted += res.RowsAffected
		}
	}
	return totalDeleted, nil
}
