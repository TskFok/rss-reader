package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func setupArticleDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.FeedAICategory{}, &models.Article{}, &models.ArticleCluster{}, &models.ArticleAIMetadata{}, &models.ArticleAIMetadataJob{}, &models.UserArticle{}))
	return db
}

func TestArticleService_List(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	items, total, err := svc.List(1, ListArticlesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Equal(t, int64(0), total)
}

func TestArticleService_MarkRead_NotFound(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	err := svc.MarkRead(1, 999)
	assert.ErrorIs(t, err, ErrArticleNotFound)
}

func TestArticleService_CleanupExpiredArticles(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	// 创建用户和订阅
	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 7}
	require.NoError(t, db.Create(&feed).Error)

	oldTime := time.Now().AddDate(0, 0, -10) // 10 天前
	newTime := time.Now().AddDate(0, 0, -3)  // 3 天前

	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "old", PublishedAt: &oldTime}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "new", PublishedAt: &newTime}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	n, err := svc.CleanupExpiredArticles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1))

	var count int64
	db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&count)
	assert.Equal(t, int64(1), count) // 只保留 3 天前的
}

func TestArticleService_CleanupExpiredArticles_ExcludesFavorited(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 7}
	require.NoError(t, db.Create(&feed).Error)

	oldTime := time.Now().AddDate(0, 0, -10)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "old", PublishedAt: &oldTime}
	require.NoError(t, db.Create(&a1).Error)

	ua := models.UserArticle{UserID: user.ID, ArticleID: a1.ID, Favorite: true}
	require.NoError(t, db.Create(&ua).Error)

	n, err := svc.CleanupExpiredArticles()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	var count int64
	db.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&count)
	assert.Equal(t, int64(1), count) // 收藏的文章不应被删除
}

func TestArticleService_ToggleFavorite(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	article := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a"}
	require.NoError(t, db.Create(&article).Error)

	// 首次收藏
	fav, err := svc.ToggleFavorite(user.ID, article.ID)
	require.NoError(t, err)
	assert.True(t, fav)

	// 取消收藏
	fav, err = svc.ToggleFavorite(user.ID, article.ID)
	require.NoError(t, err)
	assert.False(t, fav)

	// 再次收藏
	fav, err = svc.ToggleFavorite(user.ID, article.ID)
	require.NoError(t, err)
	assert.True(t, fav)
}

func TestArticleService_ToggleFavorite_NotFound(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	_, err := svc.ToggleFavorite(1, 999)
	assert.ErrorIs(t, err, ErrArticleNotFound)
}

func TestArticleService_List_Favorite(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "a1"}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "a2"}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	// 收藏 a1
	require.NoError(t, db.Create(&models.UserArticle{UserID: user.ID, ArticleID: a1.ID, Favorite: true}).Error)

	items, total, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20, Favorite: ptr(true)})
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int64(1), total)
	assert.True(t, items[0].Favorite)
	assert.Equal(t, "a1", items[0].Title)
}

func ptr(b bool) *bool {
	return &b
}

func TestArticleService_ListForSummary(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "测试订阅", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	now := time.Now()
	t1 := now.AddDate(0, 0, -5)
	t2 := now.AddDate(0, 0, -2)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "文章1", Content: "<p>内容1</p>", PublishedAt: &t1}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "文章2", Content: "内容2", PublishedAt: &t2}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	// 全部订阅、无时间限制
	items, total, err := svc.ListForSummary(user.ID, nil, nil, nil, 1, 100, "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
	assert.Equal(t, "文章2", items[0].Title)
	assert.Equal(t, "文章1", items[1].Title)
	assert.Equal(t, "测试订阅", items[0].FeedTitle)
	assert.Contains(t, items[0].Content, "内容2")
	assert.NotContains(t, items[0].Content, "<p>")

	// 指定 feed_ids
	items, total, err = svc.ListForSummary(user.ID, []uint{feed.ID}, nil, nil, 1, 100, "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)

	// 时间范围
	start := now.AddDate(0, 0, -4)
	end := now.AddDate(0, 0, -1)
	items, total, err = svc.ListForSummary(user.ID, nil, &start, &end, 1, 100, "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)
	assert.Equal(t, "文章2", items[0].Title)

	// 分页：page=2,page_size=1 应返回较早的那篇
	items, total, err = svc.ListForSummary(user.ID, nil, nil, nil, 2, 1, "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 1)
	assert.Equal(t, "文章1", items[0].Title)

	// 排序：asc=从旧到新
	items, total, err = svc.ListForSummary(user.ID, nil, nil, nil, 1, 100, "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	assert.Equal(t, "文章1", items[0].Title)
	assert.Equal(t, "文章2", items[1].Title)

	// 无文章
	start2 := now.AddDate(0, 0, -10)
	end2 := now.AddDate(0, 0, -8)
	items, total, err = svc.ListForSummary(user.ID, nil, &start2, &end2, 1, 100, "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, items)
}

func TestArticleService_List_SearchAndCluster(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "AI Feed", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "OpenAI 发布新 Agent", Content: "这是一篇关于 AI Agent 发布的文章"}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "数据库调优", Content: "介绍 MySQL 索引优化"}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	items, total, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20, Query: "Agent", Importance: 2})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Contains(t, items[0].Tags, "AI")
	assert.NotNil(t, items[0].ClusterID)
	assert.NotEmpty(t, items[0].ClusterTitle)

	clusters, err := svc.ListClusters(user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, clusters)
	assert.GreaterOrEqual(t, clusters[0].ArticleCount, 1)
}

func TestArticleService_List_DoesNotEnsureAllUserArticles(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	a1 := models.Article{FeedID: feed.ID, GUID: "g1", Title: "t1", Content: "c1"}
	a2 := models.Article{FeedID: feed.ID, GUID: "g2", Title: "t2", Content: "c2"}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	// 只拉取一页 1 条：应只为该页文章生成 metadata，而不是全量处理两篇文章
	items, total, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 1)

	var metaCount int64
	require.NoError(t, db.Model(&models.ArticleAIMetadata{}).Where("user_id = ?", user.ID).Count(&metaCount).Error)
	assert.Equal(t, int64(1), metaCount)
}

func TestArticleAIMetadataJobService_ProcessPending(t *testing.T) {
	db := setupArticleDB(t)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "FeedTitle", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)

	a := models.Article{FeedID: feed.ID, GUID: "g1", Title: "OpenAI 发布新 Agent", Content: "这是一篇关于 AI Agent 发布的文章"}
	require.NoError(t, db.Create(&a).Error)

	jobSvc := NewArticleAIMetadataJobService(db)
	require.NoError(t, jobSvc.Enqueue(user.ID, a.ID))

	n, err := jobSvc.ProcessPending(10)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	var meta models.ArticleAIMetadata
	require.NoError(t, db.Where("user_id = ? AND article_id = ?", user.ID, a.ID).First(&meta).Error)
	assert.NotEmpty(t, meta.TagsJSON)

	var cluster models.ArticleCluster
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&cluster).Error)
	assert.NotEmpty(t, cluster.ClusterKey)
}

func TestArticleAIMetadataJobService_ClaimPending_DoesNotReclaim(t *testing.T) {
	db := setupArticleDB(t)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "FeedTitle", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	a := models.Article{FeedID: feed.ID, GUID: "g1", Title: "t", Content: "c"}
	require.NoError(t, db.Create(&a).Error)

	jobSvc := NewArticleAIMetadataJobService(db)
	require.NoError(t, jobSvc.Enqueue(user.ID, a.ID))

	j1, err := jobSvc.ClaimPending(10)
	require.NoError(t, err)
	require.Len(t, j1, 1)

	j2, err := jobSvc.ClaimPending(10)
	require.NoError(t, err)
	assert.Len(t, j2, 0)
}

func TestRebuildArticleClusters_AggregatesFeedAICategories(t *testing.T) {
	db := setupArticleDB(t)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed1 := models.Feed{UserID: user.ID, URL: "http://e1", Title: "F1", UpdateIntervalMinutes: 60, ExpireDays: 0}
	feed2 := models.Feed{UserID: user.ID, URL: "http://e2", Title: "F2", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed1).Error)
	require.NoError(t, db.Create(&feed2).Error)

	require.NoError(t, db.Create(&models.FeedAICategory{UserID: user.ID, FeedID: feed1.ID, Name: "云原生"}).Error)
	require.NoError(t, db.Create(&models.FeedAICategory{UserID: user.ID, FeedID: feed2.ID, Name: "数据库"}).Error)

	a1 := models.Article{FeedID: feed1.ID, GUID: "g1", Title: "t1", Content: "c"}
	a2 := models.Article{FeedID: feed2.ID, GUID: "g2", Title: "t2", Content: "c"}
	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)

	m1 := models.ArticleAIMetadata{ArticleID: a1.ID, UserID: user.ID, ClusterKey: "kw:topic", TagsJSON: `["x"]`, TopicsJSON: `["a"]`}
	m2 := models.ArticleAIMetadata{ArticleID: a2.ID, UserID: user.ID, ClusterKey: "kw:topic", TagsJSON: `["x"]`, TopicsJSON: `["b"]`}
	require.NoError(t, db.Create(&m1).Error)
	require.NoError(t, db.Create(&m2).Error)

	require.NoError(t, rebuildArticleClusters(db, user.ID))

	var cluster models.ArticleCluster
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&cluster).Error)
	assert.Equal(t, 2, cluster.ArticleCount)
	var names []string
	require.NoError(t, json.Unmarshal([]byte(cluster.FeedAICategoriesJSON), &names))
	assert.Equal(t, []string{"云原生", "数据库"}, names)

	require.NoError(t, db.Create(&models.FeedAICategory{UserID: user.ID, FeedID: feed1.ID, Name: "安全"}).Error)
	require.NoError(t, RefreshArticleClustersFeedAICategoriesForFeed(db, user.ID, feed1.ID))
	require.NoError(t, db.First(&cluster, cluster.ID).Error)
	require.NoError(t, json.Unmarshal([]byte(cluster.FeedAICategoriesJSON), &names))
	assert.Equal(t, []string{"云原生", "安全", "数据库"}, names)
}

func TestArticleService_ListClustersPaged(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "u", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)

	// 造 3 个聚类
	c1 := models.ArticleCluster{UserID: user.ID, ClusterKey: "k1", Title: "t1", Summary: "s1", TopicsJSON: `["a"]`, ArticleCount: 10}
	c2 := models.ArticleCluster{UserID: user.ID, ClusterKey: "k2", Title: "t2", Summary: "s2", TopicsJSON: `["b"]`, ArticleCount: 9}
	c3 := models.ArticleCluster{UserID: user.ID, ClusterKey: "k3", Title: "t3", Summary: "s3", TopicsJSON: `["c"]`, ArticleCount: 8}
	require.NoError(t, db.Create(&c1).Error)
	require.NoError(t, db.Create(&c2).Error)
	require.NoError(t, db.Create(&c3).Error)

	resp, err := svc.ListClustersPaged(user.ID, ListClustersRequest{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.Total)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "t1", resp.Items[0].Title)
	assert.Equal(t, "t2", resp.Items[1].Title)

	resp2, err := svc.ListClustersPaged(user.ID, ListClustersRequest{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp2.Total)
	require.Len(t, resp2.Items, 1)
	assert.Equal(t, "t3", resp2.Items[0].Title)
}
