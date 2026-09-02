package database

import (
	"fmt"

	"github.com/tskfok/rss-reader/internal/models"
	"github.com/tskfok/rss-reader/internal/timeutil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Init 初始化数据库连接并执行迁移
func Init(dsn string) (*gorm.DB, error) {
	dsn = ensureMySQLDSNTimezone(dsn)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if isMySQLDSN(dsn) {
		if err := setMySQLSessionTimezone(db); err != nil {
			return nil, fmt.Errorf("set mysql session timezone: %w", err)
		}
	}
	if err := migrateArticleGUIDsBeforeSchema(db); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.FeedCategory{},
		&models.Feed{},
		&models.Article{},
		&models.UserArticle{},
		&models.Proxy{},
		&models.AIModel{},
		&models.AISummaryHistory{},
		&models.AISummarySchedule{},
		&models.AISummaryTemplate{},
		&models.ErrorLog{},
		&models.AppSetting{},
	); err != nil {
		return nil, err
	}
	return db, nil
}

func setMySQLSessionTimezone(db *gorm.DB) error {
	return db.Exec("SET time_zone = ?", timeutil.MySQLTimeZone).Error
}
