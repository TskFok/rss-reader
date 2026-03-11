package scheduler

import (
	"encoding/json"
	"sync"
	"time"

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
	aiModelSvc  *services.AIModelService
	historySvc  *services.SummaryHistoryService
	cron        *cron.Cron
	workers     int
}

// New 创建调度器
func New(db *gorm.DB, rssSvc *services.RSSService, articleSvc *services.ArticleService, aiModelSvc *services.AIModelService, historySvc *services.SummaryHistoryService, workers int) *Scheduler {
	if workers <= 0 {
		workers = 3
	}
	return &Scheduler{
		db:         db,
		rssSvc:     rssSvc,
		articleSvc: articleSvc,
		aiModelSvc: aiModelSvc,
		historySvc: historySvc,
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
	// 定时总结：每分钟检查一次是否到点
	_, err = s.cron.AddFunc("@every 1m", s.runSummarySchedules)
	if err != nil {
		logger.Fatalf("scheduler: %v", err)
	}
	s.cron.Start()
	logger.Info("scheduler: started, fetch every 1m, cleanup daily at 4:00, summary schedules every 1m")
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

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// runSummarySchedules 每分钟检查是否到点执行“昨天总结”
func (s *Scheduler) runSummarySchedules() {
	if s.aiModelSvc == nil || s.articleSvc == nil || s.historySvc == nil || s.db == nil {
		return
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now()
	if loc != nil {
		now = now.In(loc)
	}
	hhmm := now.Format("15:04")

	var schedules []models.AISummarySchedule
	if err := s.db.Where("enabled = 1 AND deleted_at IS NULL").Find(&schedules).Error; err != nil {
		logger.Error("scheduler: query summary schedules error: %v", err)
		return
	}
	if len(schedules) == 0 {
		return
	}

	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup
	for i := range schedules {
		sc := schedules[i]
		if sc.RunAt != hhmm {
			continue
		}
		// 今天已执行过则跳过
		if sc.LastRunAt != nil {
			last := *sc.LastRunAt
			if loc != nil {
				last = last.In(loc)
			}
			if sameDate(last, now) {
				continue
			}
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(schedule models.AISummarySchedule) {
			defer wg.Done()
			defer func() { <-sem }()

			var feedIDs []uint
			if schedule.FeedIDsJSON != "" {
				_ = json.Unmarshal([]byte(schedule.FeedIDsJSON), &feedIDs)
			}
			err := services.RunDailySummaryForYesterday(
				schedule.UserID,
				s.aiModelSvc,
				s.articleSvc,
				s.historySvc,
				schedule.AIModelID,
				feedIDs,
				schedule.PageSize,
				schedule.Order,
				now,
				loc,
			)
			if err != nil {
				logger.Error("scheduler: run summary schedule %d error: %v", schedule.ID, err)
			}
			// 更新 LastRunAt（无论成功/失败都算已尝试，避免一分钟内重复触发）
			t := time.Now()
			_ = s.db.Model(&models.AISummarySchedule{}).Where("id = ?", schedule.ID).Update("last_run_at", &t).Error
		}(sc)
	}
	wg.Wait()
}
