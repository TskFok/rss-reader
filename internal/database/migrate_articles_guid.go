package database

import (
	"strings"

	"github.com/tskfok/rss-reader/internal/models"
	"gorm.io/gorm"
)

const articleGUIDMigrationBatchSize = 500

// migrateArticleGUIDsBeforeSchema 在 AutoMigrate 收紧 guid 列之前，为旧库增加 guid_raw 并将 guid 回填为 64 位哈希。
func migrateArticleGUIDsBeforeSchema(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("articles") {
		return nil
	}
	guidRawExisted := m.HasColumn(&models.Article{}, "GUIDRaw")
	if !guidRawExisted {
		if err := db.Exec("ALTER TABLE articles ADD COLUMN guid_raw LONGTEXT NOT NULL DEFAULT ('')").Error; err != nil {
			if err2 := db.Exec("ALTER TABLE articles ADD COLUMN guid_raw LONGTEXT").Error; err2 != nil {
				return err2
			}
		}
		return migrateArticleGUIDsFull(db)
	}
	needs, err := articlesNeedCheapGUIDMigrationCheck(db)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}
	return migrateArticleGUIDsFull(db)
}

// articlesNeedCheapGUIDMigrationCheck 仅扫描 guid 列（不读 guid_raw、不算 SHA2），用于已迁移库的快速跳过。
func articlesNeedCheapGUIDMigrationCheck(db *gorm.DB) (bool, error) {
	if db.Dialector.Name() != "mysql" {
		return articlesNeedCheapGUIDMigrationCheckGo(db)
	}
	type idRow struct {
		ID uint `gorm:"column:id"`
	}
	var row idRow
	err := db.Table("articles").Unscoped().
		Select("id").
		Where(articleGUIDCheapMigrationSQL()).
		Limit(1).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// articleGUIDCheapMigrationSQL 廉价探测：仅读 guid 列，判断 sha256- 前缀或非 64 位小写十六进制哈希。
func articleGUIDCheapMigrationSQL() string {
	return `(
		(LOWER(guid) LIKE 'sha256-%' AND CHAR_LENGTH(guid) >= 71)
		OR CHAR_LENGTH(TRIM(guid)) != 64
		OR LOWER(TRIM(guid)) NOT REGEXP '^[0-9a-f]{64}$'
	)`
}

// articleGUIDNeedsMigrationSQL 精确判断仍需迁移的行（含 guid 与 guid_raw 不一致）。
func articleGUIDNeedsMigrationSQL() string {
	return `(
		(LOWER(guid) LIKE 'sha256-%' AND CHAR_LENGTH(guid) >= 71 AND LOWER(SUBSTRING(guid, 8)) REGEXP '^[0-9a-f]{64}$')
		OR (
			TRIM(guid_raw) = ''
			AND TRIM(guid) != ''
			AND (CHAR_LENGTH(TRIM(guid)) != 64 OR LOWER(TRIM(guid)) NOT REGEXP '^[0-9a-f]{64}$')
		)
		OR (
			TRIM(guid_raw) != ''
			AND guid != LOWER(SHA2(TRIM(guid_raw), 256))
		)
	)`
}

func migrateArticleGUIDsFull(db *gorm.DB) error {
	if db.Dialector.Name() == "mysql" {
		if err := migrateArticleGUIDsBulkMySQL(db); err != nil {
			return err
		}
		needs, err := articlesNeedGUIDMigrationCheck(db)
		if err != nil {
			return err
		}
		if !needs {
			return nil
		}
	}
	return migrateArticleGUIDsBatched(db)
}

func articlesNeedGUIDMigrationCheck(db *gorm.DB) (bool, error) {
	if db.Dialector.Name() != "mysql" {
		return articlesNeedGUIDMigrationCheckGo(db)
	}
	type idRow struct {
		ID uint `gorm:"column:id"`
	}
	var row idRow
	err := db.Table("articles").Unscoped().
		Select("id").
		Where(articleGUIDNeedsMigrationSQL()).
		Limit(1).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func migrateArticleGUIDsBulkMySQL(db *gorm.DB) error {
	stmts := []string{
		`UPDATE articles SET guid = LOWER(SUBSTRING(guid, 8)), guid_raw = ''
		 WHERE LOWER(guid) LIKE 'sha256-%'
		   AND CHAR_LENGTH(guid) >= 71
		   AND LOWER(SUBSTRING(guid, 8)) REGEXP '^[0-9a-f]{64}$'`,
		`UPDATE articles SET guid_raw = TRIM(guid), guid = LOWER(SHA2(TRIM(guid), 256))
		 WHERE TRIM(guid_raw) = ''
		   AND TRIM(guid) != ''
		   AND (CHAR_LENGTH(TRIM(guid)) != 64 OR LOWER(TRIM(guid)) NOT REGEXP '^[0-9a-f]{64}$')`,
		`UPDATE articles SET guid = LOWER(SHA2(TRIM(guid_raw), 256)), guid_raw = TRIM(guid_raw)
		 WHERE TRIM(guid_raw) != ''
		   AND guid != LOWER(SHA2(TRIM(guid_raw), 256))`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateArticleGUIDsBatched(db *gorm.DB) error {
	type row struct {
		ID uint   `gorm:"column:id"`
		G  string `gorm:"column:guid"`
		GR string `gorm:"column:guid_raw"`
	}

	where := ""
	if db.Dialector.Name() == "mysql" {
		where = articleGUIDNeedsMigrationSQL()
	}

	var lastID uint
	for {
		q := db.Table("articles").Unscoped().
			Select("id", "guid", "guid_raw").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(articleGUIDMigrationBatchSize)
		if where != "" {
			q = q.Where(where)
		}
		var rows []row
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, r := range rows {
			wantG, wantGR, need := normalizeArticleGUID(r.G, r.GR)
			if !need {
				continue
			}
			if err := db.Exec("UPDATE articles SET guid = ?, guid_raw = ? WHERE id = ?", wantG, wantGR, r.ID).Error; err != nil {
				return err
			}
		}
		lastID = rows[len(rows)-1].ID
		if where == "" && len(rows) < articleGUIDMigrationBatchSize {
			return nil
		}
	}
}

func articlesNeedGUIDMigrationCheckGo(db *gorm.DB) (bool, error) {
	return scanArticlesForGUIDMigration(db, func(g, gr string) bool {
		_, _, need := normalizeArticleGUID(g, gr)
		return need
	})
}

func articlesNeedCheapGUIDMigrationCheckGo(db *gorm.DB) (bool, error) {
	return scanArticlesForGUIDMigration(db, articleGUIDNeedsCheapMigration)
}

func scanArticlesForGUIDMigration(db *gorm.DB, needsMigration func(g, gr string) bool) (bool, error) {
	type row struct {
		ID uint   `gorm:"column:id"`
		G  string `gorm:"column:guid"`
		GR string `gorm:"column:guid_raw"`
	}
	var lastID uint
	for {
		var rows []row
		if err := db.Table("articles").Unscoped().
			Select("id", "guid", "guid_raw").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(articleGUIDMigrationBatchSize).
			Find(&rows).Error; err != nil {
			return false, err
		}
		if len(rows) == 0 {
			return false, nil
		}
		for _, r := range rows {
			if needsMigration(r.G, r.GR) {
				return true, nil
			}
		}
		lastID = rows[len(rows)-1].ID
		if len(rows) < articleGUIDMigrationBatchSize {
			return false, nil
		}
	}
}

func articleGUIDNeedsCheapMigration(g, gr string) bool {
	g = strings.TrimSpace(g)
	lg := strings.ToLower(g)
	if strings.HasPrefix(lg, "sha256-") && len(g) >= 7+64 {
		return true
	}
	return !models.IsArticleGUIDHash(g) || len(g) != 64
}

// normalizeArticleGUID 计算目标 guid/guid_raw；need 为 false 表示无需更新。
func normalizeArticleGUID(g, gr string) (wantG, wantGR string, need bool) {
	g = strings.TrimSpace(g)
	gr = strings.TrimSpace(gr)

	lg := strings.ToLower(g)
	if strings.HasPrefix(lg, "sha256-") && len(g) >= 7+64 {
		suffix := strings.ToLower(g[7:])
		if models.IsArticleGUIDHash(suffix) {
			return suffix, "", true
		}
	}

	raw := gr
	if raw == "" {
		if models.IsArticleGUIDHash(g) && len(g) == 64 {
			return "", "", false
		}
		raw = g
	}
	if raw == "" {
		return "", "", false
	}
	want := models.ArticleGUIDHash(raw)
	if g == want && gr == raw {
		return "", "", false
	}
	return want, raw, true
}
