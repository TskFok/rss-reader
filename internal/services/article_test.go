package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tskfok/rss-reader/internal/models"
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

func TestArticleService_MarkRead(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "reader", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	feed := models.Feed{UserID: user.ID, URL: "http://example.com", Title: "F", UpdateIntervalMinutes: 60, ExpireDays: 0}
	require.NoError(t, db.Create(&feed).Error)
	article := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g-read"), GUIDRaw: "g-read", Title: "a"}
	require.NoError(t, db.Create(&article).Error)

	require.NoError(t, svc.MarkRead(user.ID, article.ID))

	var ua models.UserArticle
	require.NoError(t, db.Where("user_id = ? AND article_id = ?", user.ID, article.ID).First(&ua).Error)
	assert.True(t, ua.ReadStatus)

	// 再次标记已读应保持幂等，不应报错。
	require.NoError(t, svc.MarkRead(user.ID, article.ID))

	var count int64
	require.NoError(t, db.Model(&models.UserArticle{}).Where("user_id = ? AND article_id = ?", user.ID, article.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
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

	a1 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "old", PublishedAt: &oldTime}
	a2 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g2"), GUIDRaw: "g2", Title: "new", PublishedAt: &newTime}
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
	a1 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "old", PublishedAt: &oldTime}
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
	article := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "a"}
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
	a1 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "a1"}
	a2 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g2"), GUIDRaw: "g2", Title: "a2"}
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
	a1 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g1"), GUIDRaw: "g1", Title: "文章1", Content: "<p>内容1</p>", PublishedAt: &t1}
	a2 := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("g2"), GUIDRaw: "g2", Title: "文章2", Content: "内容2", PublishedAt: &t2}
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

func TestArticleService_GetWithRead(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)

	user := models.User{Username: "gw", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	mid := uint(7)
	feed := models.Feed{
		UserID: user.ID, URL: "http://e.com", Title: "F", UpdateIntervalMinutes: 60,
		AIModelID: &mid, AITargetLanguage: "en",
	}
	require.NoError(t, db.Create(&feed).Error)
	a := models.Article{FeedID: feed.ID, GUID: models.ArticleGUIDHash("gx"), GUIDRaw: "gx", Title: "T"}
	require.NoError(t, db.Create(&a).Error)

	ar, err := svc.GetWithRead(user.ID, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, ar.ID)
	assert.Equal(t, "F", ar.FeedTitle)
	assert.NotNil(t, ar.FeedAIModelID)
	assert.Equal(t, uint(7), *ar.FeedAIModelID)
	assert.Equal(t, "en", ar.FeedAITargetLanguage)

	_, err = svc.GetWithRead(999, a.ID)
	assert.ErrorIs(t, err, ErrArticleNotFound)
}

func seedArticleListFixtures(t *testing.T, db *gorm.DB) (models.User, models.Feed, []models.Article) {
	t.Helper()
	user := models.User{Username: "list-user", PasswordHash: "h"}
	require.NoError(t, db.Create(&user).Error)
	mid := uint(3)
	feed := models.Feed{
		UserID: user.ID, URL: "http://example.com", Title: "Feed A",
		UpdateIntervalMinutes: 60, ExpireDays: 0,
		AIModelID: &mid, AITargetLanguage: "en",
		AITranslateEnabled: true, AIClassifyEnabled: true,
	}
	require.NoError(t, db.Create(&feed).Error)

	now := time.Now()
	tOld := now.Add(-48 * time.Hour)
	tNew := now.Add(-1 * time.Hour)
	tMid := now.Add(-2 * time.Hour)
	articles := []models.Article{
		{
			FeedID: feed.ID, GUID: models.ArticleGUIDHash("g-old-read"), GUIDRaw: "g-old-read",
			Title: "old-read", Content: "LONGTEXT content old", ContentTranslated: "LONGTEXT translated old",
			PublishedAt: &tOld,
		},
		{
			FeedID: feed.ID, GUID: models.ArticleGUIDHash("g-new-unread"), GUIDRaw: "g-new-unread",
			Title: "new-unread", Content: "LONGTEXT content new", ContentTranslated: "LONGTEXT translated new",
			PublishedAt: &tNew,
		},
		{
			FeedID: feed.ID, GUID: models.ArticleGUIDHash("g-fav-unread"), GUIDRaw: "g-fav-unread",
			Title: "fav-unread", Content: "LONGTEXT content fav", ContentTranslated: "LONGTEXT translated fav",
			PublishedAt: &tMid,
		},
	}
	for i := range articles {
		require.NoError(t, db.Create(&articles[i]).Error)
	}
	require.NoError(t, db.Create(&models.UserArticle{
		UserID: user.ID, ArticleID: articles[0].ID, ReadStatus: true,
	}).Error)
	require.NoError(t, db.Create(&models.UserArticle{
		UserID: user.ID, ArticleID: articles[2].ID, Favorite: true,
	}).Error)
	return user, feed, articles
}

func assertListItemNoLongtext(t *testing.T, item ArticleListItem) {
	t.Helper()
	b, err := json.Marshal(item)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))
	assert.NotContains(t, m, "content")
	assert.NotContains(t, m, "content_translated")
	assert.NotContains(t, m, "guid_raw")
}

func TestArticleService_List_ExcludesLongtextFields(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)
	user, _, _ := seedArticleListFixtures(t, db)

	items, total, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, items, 3)
	for _, item := range items {
		assertListItemNoLongtext(t, item)
	}
}

func TestArticleService_List_UnreadFirstOrdering(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)
	user, _, articles := seedArticleListFixtures(t, db)

	items, _, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, items, 3)
	// 未读优先，同组内按 published_at DESC
	assert.Equal(t, articles[1].ID, items[0].ID)
	assert.Equal(t, articles[2].ID, items[1].ID)
	assert.Equal(t, articles[0].ID, items[2].ID)
	assert.False(t, items[0].Read)
	assert.False(t, items[1].Read)
	assert.True(t, items[2].Read)
}

func TestArticleService_List_ReadFavoriteFilters(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)
	user, _, articles := seedArticleListFixtures(t, db)

	readTrue := true
	readFalse := false
	favTrue := true

	items, total, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20, Read: &readTrue})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, articles[0].ID, items[0].ID)
	assert.True(t, items[0].Read)

	items, total, err = svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20, Read: &readFalse})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	items, total, err = svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20, Favorite: &favTrue})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, articles[2].ID, items[0].ID)
	assert.True(t, items[0].Favorite)
}

func TestArticleService_List_ReadAndFavoriteCombined(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)
	user, _, articles := seedArticleListFixtures(t, db)

	readFalse := false
	favTrue := true
	items, total, err := svc.List(user.ID, ListArticlesRequest{
		Page: 1, PageSize: 20, Read: &readFalse, Favorite: &favTrue,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, articles[2].ID, items[0].ID)
	assert.False(t, items[0].Read)
	assert.True(t, items[0].Favorite)
}

func TestArticleService_List_JoinMapsFeedAndUserArticle(t *testing.T) {
	db := setupArticleDB(t)
	svc := NewArticleService(db)
	user, feed, articles := seedArticleListFixtures(t, db)

	items, _, err := svc.List(user.ID, ListArticlesRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	byID := make(map[uint]ArticleListItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	fav := byID[articles[2].ID]
	assert.Equal(t, feed.Title, fav.FeedTitle)
	assert.True(t, fav.FeedAITranslateEnabled)
	assert.True(t, fav.FeedAIClassifyEnabled)
	require.NotNil(t, fav.FeedAIModelID)
	assert.Equal(t, uint(3), *fav.FeedAIModelID)
	assert.Equal(t, "en", fav.FeedAITargetLanguage)
	assert.True(t, fav.Favorite)
	assert.False(t, fav.Read)
}
