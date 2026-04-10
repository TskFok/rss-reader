package services

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
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
	FeedID        *uint  `form:"feed_id"`
	Read          *bool  `form:"read"`
	Favorite      *bool  `form:"favorite"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
	Query         string `form:"q"`
	TagIDs        string `form:"tag_ids"`
	TopicIDs      string `form:"topic_ids"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
	Importance    int    `form:"importance"`
	ClusterID     *uint  `form:"cluster_id"`
	HasAIMetadata *bool  `form:"has_ai_metadata"`
}

// ArticleWithRead 带阅读状态与收藏的文章
type ArticleWithRead struct {
	models.Article
	Read         bool     `json:"read"`
	Favorite     bool     `json:"favorite"`
	FeedTitle    string   `json:"feed_title"`
	AISummary    string   `json:"ai_summary"`
	Tags         []string `json:"tags"`
	Topics       []string `json:"topics"`
	Keywords     []string `json:"keywords"`
	Entities     []string `json:"entities"`
	Language     string   `json:"language"`
	Sentiment    string   `json:"sentiment"`
	Importance   int      `json:"importance"`
	ClusterID    *uint    `json:"cluster_id"`
	ClusterTitle string   `json:"cluster_title"`
}

// List 获取用户可见的文章列表（通过 feed 归属）
func (s *ArticleService) List(userID uint, req ListArticlesRequest) ([]ArticleWithRead, int64, error) {
	// 如果请求包含依赖 AI 元数据的筛选条件，但又不是显式只看“已有元数据”的文章，
	// 则需要先在筛选范围内补齐元数据，否则 where aim.* 会把结果全过滤掉。
	needsAIMetadata :=
		strings.TrimSpace(req.Query) != "" ||
			strings.TrimSpace(req.TagIDs) != "" ||
			strings.TrimSpace(req.TopicIDs) != "" ||
			req.Importance > 0 ||
			req.ClusterID != nil ||
			req.HasAIMetadata != nil
	if needsAIMetadata && !(req.HasAIMetadata != nil && *req.HasAIMetadata) {
		// 仅在当前筛选范围（按 feed_id）内补齐，避免全量扫描全部文章
		if err := ensureArticleMetadataForScope(s.db, userID, req.FeedID); err != nil {
			return nil, 0, err
		}
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	needJoinAIMetadata :=
		strings.TrimSpace(req.Query) != "" ||
			strings.TrimSpace(req.TagIDs) != "" ||
			strings.TrimSpace(req.TopicIDs) != "" ||
			req.Importance > 0 ||
			req.ClusterID != nil ||
			req.HasAIMetadata != nil

	q := s.db.Model(&models.Article{}).
		Joins("JOIN feeds ON feeds.id = articles.feed_id AND feeds.deleted_at IS NULL").
		Where("feeds.user_id = ?", userID)
	if needJoinAIMetadata {
		q = q.Joins("LEFT JOIN article_ai_metadata aim ON aim.article_id = articles.id AND aim.deleted_at IS NULL")
	}
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
	if query := strings.TrimSpace(req.Query); query != "" {
		like := "%" + query + "%"
		q = q.Where(
			"articles.title LIKE ? OR articles.content LIKE ? OR feeds.title LIKE ? OR aim.summary LIKE ?",
			like, like, like, like,
		)
	}
	if tags := splitCSV(req.TagIDs); len(tags) > 0 {
		for _, tag := range tags {
			q = q.Where("aim.tags_json LIKE ?", "%"+tag+"%")
		}
	}
	if topics := splitCSV(req.TopicIDs); len(topics) > 0 {
		for _, topic := range topics {
			q = q.Where("aim.topics_json LIKE ?", "%"+topic+"%")
		}
	}
	if req.Importance > 0 {
		q = q.Where("aim.importance >= ?", req.Importance)
	}
	if req.ClusterID != nil {
		q = q.Where("aim.cluster_id = ?", *req.ClusterID)
	}
	if req.HasAIMetadata != nil {
		if *req.HasAIMetadata {
			q = q.Where("aim.id IS NOT NULL")
		} else {
			q = q.Where("aim.id IS NULL")
		}
	}
	if start, end := parseDateRange(req.StartDate, req.EndDate); start != nil || end != nil {
		if start != nil {
			q = q.Where("COALESCE(articles.published_at, articles.created_at) >= ?", *start)
		}
		if end != nil {
			q = q.Where("COALESCE(articles.published_at, articles.created_at) <= ?", *end)
		}
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

	// 仅对当前页文章按需生成 AI 元数据与聚类，避免每次列表请求都全量扫描用户全部文章。
	if err := ensureArticleMetadataForArticles(s.db, userID, articles); err != nil {
		return nil, 0, err
	}

	ids := make([]uint, len(articles))
	for i := range articles {
		ids[i] = articles[i].ID
	}
	var uas []models.UserArticle
	s.db.Model(&models.UserArticle{}).
		Select("article_id", "read_status", "favorite").
		Where("user_id = ? AND article_id IN ?", userID, ids).
		Find(&uas)
	var metas []models.ArticleAIMetadata
	// 只取列表渲染所需字段，减少扫描/传输开销
	s.db.
		Select("article_id, cluster_id, summary, tags_json, topics_json, keywords_json, entities_json, language, sentiment, importance").
		Where("user_id = ? AND article_id IN ?", userID, ids).
		Find(&metas)
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
	metaMap := make(map[uint]models.ArticleAIMetadata, len(metas))
	for _, meta := range metas {
		metaMap[meta.ArticleID] = meta
	}

	// 仅查询当前页文章涉及到的 clusters，避免每次列表都全量拉取该用户所有 clusters
	clusterIDs := make([]uint, 0, len(metas))
	seenCluster := make(map[uint]struct{}, len(metas))
	for _, meta := range metas {
		if meta.ClusterID == nil {
			continue
		}
		if _, ok := seenCluster[*meta.ClusterID]; ok {
			continue
		}
		seenCluster[*meta.ClusterID] = struct{}{}
		clusterIDs = append(clusterIDs, *meta.ClusterID)
	}
	var clusters []models.ArticleCluster
	if len(clusterIDs) > 0 {
		s.db.Model(&models.ArticleCluster{}).
			Select("id", "title").
			Where("user_id = ? AND id IN ?", userID, clusterIDs).
			Find(&clusters)
	}
	clusterTitleMap := make(map[uint]string, len(clusters))
	for _, cluster := range clusters {
		clusterTitleMap[cluster.ID] = cluster.Title
	}
	result := make([]ArticleWithRead, len(articles))
	for i, a := range articles {
		feedTitle := ""
		if a.Feed.ID != 0 {
			feedTitle = a.Feed.Title
		}
		meta := metaMap[a.ID]
		clusterTitle := ""
		if meta.ClusterID != nil {
			clusterTitle = clusterTitleMap[*meta.ClusterID]
		}
		result[i] = ArticleWithRead{
			Article:      a,
			Read:         readMap[a.ID],
			Favorite:     favMap[a.ID],
			FeedTitle:    feedTitle,
			AISummary:    meta.Summary,
			Tags:         decodeJSONSlice(meta.TagsJSON),
			Topics:       decodeJSONSlice(meta.TopicsJSON),
			Keywords:     decodeJSONSlice(meta.KeywordsJSON),
			Entities:     decodeJSONSlice(meta.EntitiesJSON),
			Language:     meta.Language,
			Sentiment:    meta.Sentiment,
			Importance:   meta.Importance,
			ClusterID:    meta.ClusterID,
			ClusterTitle: clusterTitle,
		}
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

// ArticleForSummary 用于 AI 总结的文章摘要
type ArticleForSummary struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	FeedTitle   string `json:"feed_title"`
	PublishedAt string `json:"published_at"`
}

type SummaryArticleFilter struct {
	Query         string
	MinImportance int
}

// ListForSummary 获取指定订阅、时间范围内的文章，用于 AI 总结（支持分页）
// feedIDs 为空表示全部订阅；返回 items 与 total（分页前总数）
// order: "desc"(默认)=从新到旧，"asc"=从旧到新
func (s *ArticleService) ListForSummary(userID uint, feedIDs []uint, startTime, endTime *time.Time, page, pageSize int, order string) ([]ArticleForSummary, int64, error) {
	return s.ListForSummaryWithFilters(userID, feedIDs, startTime, endTime, page, pageSize, order, SummaryArticleFilter{})
}

func (s *ArticleService) ListForSummaryWithFilters(userID uint, feedIDs []uint, startTime, endTime *time.Time, page, pageSize int, order string, filter SummaryArticleFilter) ([]ArticleForSummary, int64, error) {
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
	if strings.TrimSpace(filter.Query) != "" || filter.MinImportance > 0 {
		q = q.Joins("LEFT JOIN article_ai_metadata aim ON aim.article_id = articles.id AND aim.deleted_at IS NULL")
	}
	if len(feedIDs) > 0 {
		q = q.Where("articles.feed_id IN ?", feedIDs)
	}
	if startTime != nil {
		q = q.Where("COALESCE(articles.published_at, articles.created_at) >= ?", *startTime)
	}
	if endTime != nil {
		q = q.Where("COALESCE(articles.published_at, articles.created_at) <= ?", *endTime)
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + query + "%"
		q = q.Where(
			"articles.title LIKE ? OR articles.content LIKE ? OR feeds.title LIKE ? OR aim.summary LIKE ? OR aim.keywords_json LIKE ?",
			like, like, like, like, like,
		)
	}
	if filter.MinImportance > 0 {
		if filter.MinImportance > 5 {
			filter.MinImportance = 5
		}
		q = q.Where("aim.importance >= ?", filter.MinImportance)
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

func (s *ArticleService) ListClusters(userID uint) ([]ArticleClusterItem, error) {
	return listArticleClusters(s.db, userID)
}

type ListClustersRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type ListClustersResponse struct {
	Items    []ArticleClusterItem `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

func (s *ArticleService) ListClustersPaged(userID uint, req ListClustersRequest) (ListClustersResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := listArticleClustersPaged(s.db, userID, page, pageSize)
	if err != nil {
		return ListClustersResponse{}, err
	}
	return ListClustersResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

type ArticleTopicItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListTopics 聚合用户文章元数据里的 topics 词表（按出现次数排序）。
// limit 控制参与聚合的最近元数据条数，避免大用户全量扫描；<=0 则使用默认值 2000。
func (s *ArticleService) ListTopics(userID uint, limit int) ([]ArticleTopicItem, error) {
	if limit <= 0 || limit > 20000 {
		limit = 2000
	}

	// 只取需要的列，且按 id 倒序取“最近一段”元数据
	type row struct {
		TopicsJSON string
	}
	var rows []row
	if err := s.db.Model(&models.ArticleAIMetadata{}).
		Select("topics_json").
		Where("user_id = ? AND topics_json IS NOT NULL AND topics_json <> ''", userID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]int, 64)
	for _, r := range rows {
		raw := strings.TrimSpace(r.TopicsJSON)
		if raw == "" {
			continue
		}
		var topics []string
		if err := json.Unmarshal([]byte(raw), &topics); err != nil {
			continue
		}
		seen := make(map[string]struct{}, len(topics))
		for _, t := range topics {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			counts[t]++
		}
	}

	out := make([]ArticleTopicItem, 0, len(counts))
	for name, c := range counts {
		out = append(out, ArticleTopicItem{Name: name, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	return out, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseDateRange(startDate, endDate string) (*time.Time, *time.Time) {
	var start, end *time.Time
	if strings.TrimSpace(startDate) != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			start = &t
		}
	}
	if strings.TrimSpace(endDate) != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(24*time.Hour - time.Second)
			end = &t
		}
	}
	return start, end
}

func decodeJSONSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
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
