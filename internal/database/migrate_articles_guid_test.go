package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

func TestNormalizeArticleGUID(t *testing.T) {
	hashHello := models.ArticleGUIDHash("hello")
	prefixed := "sha256-" + hashHello

	tests := []struct {
		name    string
		g, gr   string
		wantG   string
		wantGR  string
		need    bool
	}{
		{"already hashed with raw", hashHello, "hello", "", "", false},
		{"hash only no raw", hashHello, "", "", "", false},
		{"plain guid", "hello", "", hashHello, "hello", true},
		{"sha256 prefix", prefixed, "", hashHello, "", true},
		{"sha256 prefix keeps raw cleared", prefixed, "legacy", hashHello, "", true},
		{"mismatch raw", hashHello, "other", models.ArticleGUIDHash("other"), "other", true},
		{"empty guid", "", "", "", "", false},
		{"whitespace guid", "  hello  ", "  ", hashHello, "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotG, gotGR, need := normalizeArticleGUID(tt.g, tt.gr)
			assert.Equal(t, tt.need, need)
			if !tt.need {
				return
			}
			assert.Equal(t, tt.wantG, gotG)
			assert.Equal(t, tt.wantGR, gotGR)
		})
	}
}

func TestMigrateArticleGUIDsBeforeSchema_sqlite(t *testing.T) {
	db := openSQLiteTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.Feed{}, &models.Article{}))

	feed := models.Feed{Title: "f", URL: "https://example.com/rss"}
	require.NoError(t, db.Create(&feed).Error)

	hashHello := models.ArticleGUIDHash("hello")
	prefixed := "sha256-" + hashHello

	articles := []models.Article{
		{FeedID: feed.ID, GUID: "hello", GUIDRaw: "", Title: "legacy plain"},
		{FeedID: feed.ID, GUID: prefixed, GUIDRaw: "x", Title: "legacy prefixed"},
		{FeedID: feed.ID, GUID: hashHello, GUIDRaw: "hello", Title: "already ok"},
		{FeedID: feed.ID, GUID: hashHello, GUIDRaw: "", Title: "hash only empty raw"},
	}
	require.NoError(t, db.Create(&articles).Error)

	require.NoError(t, migrateArticleGUIDsBeforeSchema(db))

	var migrated []models.Article
	require.NoError(t, db.Order("id").Find(&migrated).Error)
	require.Len(t, migrated, 4)

	assert.Equal(t, hashHello, migrated[0].GUID)
	assert.Equal(t, "hello", migrated[0].GUIDRaw)
	assert.Equal(t, hashHello, migrated[1].GUID)
	assert.Equal(t, "", migrated[1].GUIDRaw)
	assert.Equal(t, hashHello, migrated[2].GUID)
	assert.Equal(t, "hello", migrated[2].GUIDRaw)
	assert.Equal(t, hashHello, migrated[3].GUID)
	assert.Equal(t, "", migrated[3].GUIDRaw)

	// 已迁移库再次启动应快速跳过（廉价探测：guid 均为 64 位哈希）
	require.NoError(t, migrateArticleGUIDsBeforeSchema(db))
}

func TestMigrateArticleGUIDsBeforeSchema_addsGuidRawColumn(t *testing.T) {
	db := openSQLiteTestDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE articles (
		id INTEGER PRIMARY KEY,
		feed_id INTEGER NOT NULL,
		guid TEXT NOT NULL,
		title TEXT,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO articles (feed_id, guid, title) VALUES (1, 'item-1', 't')`).Error)

	require.NoError(t, migrateArticleGUIDsBeforeSchema(db))

	var row struct {
		GUID    string
		GUIDRaw string
	}
	require.NoError(t, db.Table("articles").Select("guid", "guid_raw").Take(&row).Error)
	assert.Equal(t, models.ArticleGUIDHash("item-1"), row.GUID)
	assert.Equal(t, "item-1", row.GUIDRaw)
}

func openSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	return db
}
