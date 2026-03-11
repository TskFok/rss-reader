package services

import (
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
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.FeedCategory{}, &models.Feed{}, &models.Article{}, &models.UserArticle{}))
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
	items, err := svc.ListForSummary(user.ID, nil, nil, nil, 100)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "文章2", items[0].Title)
	assert.Equal(t, "文章1", items[1].Title)
	assert.Equal(t, "测试订阅", items[0].FeedTitle)
	assert.Contains(t, items[0].Content, "内容2")
	assert.NotContains(t, items[0].Content, "<p>")

	// 指定 feed_ids
	items, err = svc.ListForSummary(user.ID, []uint{feed.ID}, nil, nil, 100)
	require.NoError(t, err)
	assert.Len(t, items, 2)

	// 时间范围
	start := now.AddDate(0, 0, -4)
	end := now.AddDate(0, 0, -1)
	items, err = svc.ListForSummary(user.ID, nil, &start, &end, 100)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "文章2", items[0].Title)

	// 无文章
	start2 := now.AddDate(0, 0, -10)
	end2 := now.AddDate(0, 0, -8)
	items, err = svc.ListForSummary(user.ID, nil, &start2, &end2, 100)
	require.NoError(t, err)
	assert.Empty(t, items)
}
