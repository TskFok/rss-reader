package database

import (
	"strings"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

// migrateArticleGUIDsBeforeSchema 在 AutoMigrate 收紧 guid 列之前，为旧库增加 guid_raw 并将 guid 回填为 64 位哈希。
func migrateArticleGUIDsBeforeSchema(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("articles") {
		return nil
	}
	if !m.HasColumn(&models.Article{}, "GUIDRaw") {
		if err := db.Exec("ALTER TABLE articles ADD COLUMN guid_raw LONGTEXT NOT NULL DEFAULT ('')").Error; err != nil {
			if err2 := db.Exec("ALTER TABLE articles ADD COLUMN guid_raw LONGTEXT").Error; err2 != nil {
				return err2
			}
		}
	}

	type row struct {
		ID uint   `gorm:"column:id"`
		G  string `gorm:"column:guid"`
		GR string `gorm:"column:guid_raw"`
	}
	var rows []row
	if err := db.Table("articles").Unscoped().Select("id", "guid", "guid_raw").Find(&rows).Error; err != nil {
		return err
	}

	for _, r := range rows {
		g := strings.TrimSpace(r.G)
		gr := strings.TrimSpace(r.GR)

		lg := strings.ToLower(g)
		if strings.HasPrefix(lg, "sha256-") && len(g) >= 7+64 {
			suffix := strings.ToLower(g[7:])
			if models.IsArticleGUIDHash(suffix) {
				if err := db.Exec("UPDATE articles SET guid = ?, guid_raw = ? WHERE id = ?", suffix, "", r.ID).Error; err != nil {
					return err
				}
				continue
			}
		}

		raw := gr
		if raw == "" {
			if models.IsArticleGUIDHash(g) && len(g) == 64 {
				continue
			}
			raw = g
		}
		if raw == "" {
			continue
		}
		want := models.ArticleGUIDHash(raw)
		if g == want && gr == raw {
			continue
		}
		if err := db.Exec("UPDATE articles SET guid = ?, guid_raw = ? WHERE id = ?", want, raw, r.ID).Error; err != nil {
			return err
		}
	}
	return nil
}
