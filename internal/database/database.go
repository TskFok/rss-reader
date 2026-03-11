package database

import (
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Init 初始化数据库连接并执行迁移
func Init(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
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
	); err != nil {
		return nil, err
	}
	return db, nil
}
