package services

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

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
	Read                   bool   `json:"read"`
	Favorite               bool   `json:"favorite"`
	FeedTitle              string `json:"feed_title"`
	FeedAITranslateEnabled bool   `json:"feed_ai_translate_enabled"`
	FeedAIClassifyEnabled  bool   `json:"feed_ai_classify_enabled"`
	FeedAIModelID          *uint  `json:"feed_ai_model_id"`
	FeedAITargetLanguage   string `json:"feed_ai_target_language"`
}

// ArticleListItem 文章列表项（不含 content、content_translated、guid_raw）
type ArticleListItem struct {
	ID                     uint       `json:"id" gorm:"column:id"`
	FeedID                 uint       `json:"feed_id" gorm:"column:feed_id"`
	GUID                   string     `json:"guid" gorm:"column:guid"`
	Title                  string     `json:"title" gorm:"column:title"`
	Link                   string     `json:"link" gorm:"column:link"`
	AIProcessStatus        string     `json:"ai_process_status" gorm:"column:ai_process_status"`
	AILastError            string     `json:"ai_last_error" gorm:"column:ai_last_error"`
	AICategory             string     `json:"ai_category" gorm:"column:ai_category"`
	AICategoryTranslated   string     `json:"ai_category_translated" gorm:"column:ai_category_translated"`
	TitleTranslated        string     `json:"title_translated" gorm:"column:title_translated"`
	PublishedAt            *time.Time `json:"published_at" gorm:"column:published_at"`
	CreatedAt              time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt              time.Time  `json:"updated_at" gorm:"column:updated_at"`
	Read                   bool       `json:"read" gorm:"column:read"`
	Favorite               bool       `json:"favorite" gorm:"column:favorite"`
	FeedTitle              string     `json:"feed_title" gorm:"column:feed_title"`
	FeedAITranslateEnabled bool       `json:"feed_ai_translate_enabled" gorm:"column:feed_ai_translate_enabled"`
	FeedAIClassifyEnabled  bool       `json:"feed_ai_classify_enabled" gorm:"column:feed_ai_classify_enabled"`
	FeedAIModelID          *uint      `json:"feed_ai_model_id" gorm:"column:feed_ai_model_id"`
	FeedAITargetLanguage   string     `json:"feed_ai_target_language" gorm:"column:feed_ai_target_language"`
}

var (
	articleListArticleColumns = []string{
		"articles.id",
		"articles.feed_id",
		"articles.guid",
		"articles.title",
		"articles.link",
		"articles.ai_process_status",
		"articles.ai_last_error",
		"articles.ai_category",
		"articles.ai_category_translated",
		"articles.title_translated",
		"articles.published_at",
		"articles.created_at",
		"articles.updated_at",
	}
	articleListFeedColumns = []string{
		"feeds.title AS feed_title",
		"feeds.ai_translate_enabled AS feed_ai_translate_enabled",
		"feeds.ai_classify_enabled AS feed_ai_classify_enabled",
		"feeds.ai_model_id AS feed_ai_model_id",
		"feeds.ai_target_language AS feed_ai_target_language",
	}
	articleListUAColumns = []string{
		"COALESCE(ua.read_status, 0) AS read",
		"COALESCE(ua.favorite, 0) AS favorite",
	}
)

func articleListSelectColumns() []string {
	cols := make([]string, 0, len(articleListArticleColumns)+len(articleListFeedColumns)+len(articleListUAColumns))
	cols = append(cols, articleListArticleColumns...)
	cols = append(cols, articleListFeedColumns...)
	cols = append(cols, articleListUAColumns...)
	return cols
}

// List 获取用户可见的文章列表（通过 feed 归属）
func (s *ArticleService) List(userID uint, req ListArticlesRequest) ([]ArticleListItem, int64, error) {
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
		Joins("LEFT JOIN user_articles ua ON ua.article_id = articles.id AND ua.user_id = ?", userID).
		Where("feeds.user_id = ?", userID)
	if req.FeedID != nil {
		q = q.Where("articles.feed_id = ?", *req.FeedID)
	}
	if req.Read != nil {
		if *req.Read {
			q = q.Where("ua.read_status = ?", true)
		} else {
			q = q.Where("ua.id IS NULL OR ua.read_status = ?", false)
		}
	}
	if req.Favorite != nil && *req.Favorite {
		q = q.Where("ua.favorite = ?", true)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	orderClause := "articles.published_at DESC, articles.created_at DESC"
	if req.Read == nil {
		orderClause = "COALESCE(ua.read_status, 0) ASC, " + orderClause
	}
	var result []ArticleListItem
	if err := q.Select(articleListSelectColumns()).
		Order(orderClause).
		Offset(offset).Limit(pageSize).
		Scan(&result).Error; err != nil {
		return nil, 0, err
	}
	if result == nil {
		return []ArticleListItem{}, total, nil
	}
	return result, total, nil
}

// GetWithRead 单篇详情（校验订阅归属），用于手动 AI 等接口返回最新数据
func (s *ArticleService) GetWithRead(userID uint, articleID uint) (ArticleWithRead, error) {
	var article models.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ArticleWithRead{}, ErrArticleNotFound
		}
		return ArticleWithRead{}, err
	}
	var feed models.Feed
	if err := s.db.First(&feed, article.FeedID).Error; err != nil {
		return ArticleWithRead{}, ErrArticleNotFound
	}
	if feed.UserID != userID {
		return ArticleWithRead{}, ErrArticleNotFound
	}
	article.Feed = feed
	var ua models.UserArticle
	_ = s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&ua).Error
	feedTitle := ""
	if article.Feed.ID != 0 {
		feedTitle = article.Feed.Title
	}
	return ArticleWithRead{
		Article:                article,
		Read:                   ua.ReadStatus,
		Favorite:               ua.Favorite,
		FeedTitle:              feedTitle,
		FeedAITranslateEnabled: article.Feed.AITranslateEnabled,
		FeedAIClassifyEnabled:  article.Feed.AIClassifyEnabled,
		FeedAIModelID:          article.Feed.AIModelID,
		FeedAITargetLanguage:   article.Feed.AITargetLanguage,
	}, nil
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

// ArticleForSummary 用于 AI 总结的文章摘要
type ArticleForSummary struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	FeedTitle   string `json:"feed_title"`
	PublishedAt string `json:"published_at"`
}

// ListForSummary 获取指定订阅、时间范围内的文章，用于 AI 总结（支持分页）
// feedIDs 为空表示全部订阅；返回 items 与 total（分页前总数）
// order: "desc"(默认)=从新到旧，"asc"=从旧到新
func (s *ArticleService) ListForSummary(userID uint, feedIDs []uint, startTime, endTime *time.Time, page, pageSize int, order string) ([]ArticleForSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	order = strings.ToLower(strings.TrimSpace(order))
	orderClause := "articles.published_at DESC, articles.created_at DESC"
	if order == "asc" {
		orderClause = "articles.published_at ASC, articles.created_at ASC"
	}
	q := s.db.Model(&models.Article{}).
		Joins("JOIN feeds ON feeds.id = articles.feed_id AND feeds.deleted_at IS NULL").
		Where("feeds.user_id = ?", userID)
	if len(feedIDs) > 0 {
		q = q.Where("articles.feed_id IN ?", feedIDs)
	}
	if startTime != nil {
		q = q.Where("COALESCE(articles.published_at, articles.created_at) >= ?", *startTime)
	}
	if endTime != nil {
		q = q.Where("COALESCE(articles.published_at, articles.created_at) <= ?", *endTime)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var articles []models.Article
	offset := (page - 1) * pageSize
	if err := q.Order(orderClause).
		Offset(offset).Limit(pageSize).
		Preload("Feed").
		Find(&articles).Error; err != nil {
		return nil, 0, err
	}
	result := make([]ArticleForSummary, 0, len(articles))
	for _, a := range articles {
		feedTitle := ""
		if a.Feed.ID != 0 {
			feedTitle = a.Feed.Title
		}
		pubStr := ""
		if a.PublishedAt != nil {
			pubStr = a.PublishedAt.Format("2006-01-02 15:04")
		} else {
			pubStr = a.CreatedAt.Format("2006-01-02 15:04")
		}
		content := htmlTagRe.ReplaceAllString(a.Content, " ")
		content = strings.TrimSpace(strings.Join(strings.Fields(content), " "))
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		result = append(result, ArticleForSummary{
			Title:       a.Title,
			Content:     content,
			FeedTitle:   feedTitle,
			PublishedAt: pubStr,
		})
	}
	return result, total, nil
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
