package scheduler

import (
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/ushopal/rss-reader/internal/logger"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/services"
	"gorm.io/gorm"
)

// Scheduler 定时更新调度器
type Scheduler struct {
	db          *gorm.DB
	rssSvc      *services.RSSService
	articleSvc  *services.ArticleService
	cron        *cron.Cron
	workers     int
}

// New 创建调度器
func New(db *gorm.DB, rssSvc *services.RSSService, articleSvc *services.ArticleService, workers int) *Scheduler {
	if workers <= 0 {
		workers = 3
	}
	return &Scheduler{
		db:         db,
		rssSvc:     rssSvc,
		articleSvc: articleSvc,
		cron:       cron.New(),
		workers:    workers,
	}
}

// Start 启动调度：每分钟抓取，每天清理过期文章
func (s *Scheduler) Start() {
	_, err := s.cron.AddFunc("@every 1m", s.runFetch)
	if err != nil {
		logger.Fatalf("scheduler: %v", err)
	}
	_, err = s.cron.AddFunc("0 4 * * *", s.runCleanup) // 每天 4:00 执行
	if err != nil {
		logger.Fatalf("scheduler: %v", err)
	}
	s.cron.Start()
	logger.Info("scheduler: started, fetch every 1m, cleanup daily at 4:00")
}

// Stop 停止调度
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// runFetch 执行抓取任务
func (s *Scheduler) runFetch() {
	var feeds []models.Feed
	err := s.db.Preload("Proxy").Where(
		"deleted_at IS NULL AND (last_fetched_at IS NULL OR DATE_ADD(last_fetched_at, INTERVAL update_interval_minutes MINUTE) <= NOW())",
	).Find(&feeds).Error
	if err != nil {
		logger.Error("scheduler: query feeds error: %v", err)
		return
	}
	if len(feeds) == 0 {
		return
	}
	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup
	for i := range feeds {
		wg.Add(1)
		sem <- struct{}{}
		go func(f *models.Feed) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.rssSvc.FetchFeed(f); err != nil {
				logger.Error("scheduler: fetch feed %d (%s) error: %v", f.ID, f.URL, err)
			}
		}(&feeds[i])
	}
	wg.Wait()
}

// runCleanup 每日清理过期文章
func (s *Scheduler) runCleanup() {
	if s.articleSvc == nil {
		return
	}
	n, err := s.articleSvc.CleanupExpiredArticles()
	if err != nil {
		logger.Error("scheduler: cleanup expired articles error: %v", err)
		return
	}
	if n > 0 {
		logger.Info("scheduler: cleanup expired articles, deleted %d records", n)
	}
}
