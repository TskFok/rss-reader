package services

import (
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/logger"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

// 补漏批处理仅读取处理所需列，避免拉取 guid_raw、content_translated 全文等大字段（content 除外）。
var (
	articleBackfillClassifySelect = []string{
		"id", "feed_id", "title", "content",
		"title_translated", "content_translated", "ai_process_status",
	}
	articleBackfillTranslateSelect = []string{
		"id", "feed_id", "title", "content", "ai_category",
		"title_translated", "content_translated", "ai_process_status",
	}
)

func classifyBackfillFeedIDs(db *gorm.DB) ([]uint, error) {
	var feedIDs []uint
	err := db.Model(&models.Feed{}).
		Where("ai_classify_enabled = ?", true).
		Where("ai_model_id IS NOT NULL AND ai_model_id > ?", 0).
		Pluck("id", &feedIDs).Error
	return feedIDs, err
}

func translateBackfillFeedIDs(db *gorm.DB) (feedIDs, classifyFeedIDs []uint, err error) {
	type feedRow struct {
		ID                uint
		AIClassifyEnabled bool
	}
	var rows []feedRow
	err = db.Model(&models.Feed{}).
		Select("id", "ai_classify_enabled").
		Where("ai_translate_enabled = ?", true).
		Where("ai_target_language <> ''").
		Where("ai_model_id IS NOT NULL AND ai_model_id > ?", 0).
		Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		feedIDs = append(feedIDs, r.ID)
		if r.AIClassifyEnabled {
			classifyFeedIDs = append(classifyFeedIDs, r.ID)
		}
	}
	return feedIDs, classifyFeedIDs, nil
}

func applyBackfillStalePendingFilter(q *gorm.DB, stalePendingCutoff time.Time) *gorm.DB {
	return q.Where("NOT (ai_process_status = ? AND updated_at > ?)", models.AIProcessPending, stalePendingCutoff)
}

func applyTranslateBackfillArticleFilters(q *gorm.DB, classifyFeedIDs []uint) *gorm.DB {
	q = q.Where(
		"(content_translated = '' OR content_translated IS NULL OR ai_process_status IN ?)",
		[]string{models.AIProcessFailed, models.AIProcessPending},
	)
	if len(classifyFeedIDs) > 0 {
		q = q.Where(
			"(feed_id NOT IN ? OR (ai_category <> '' AND ai_category IS NOT NULL))",
			classifyFeedIDs,
		)
	}
	return q
}

// BackfillClassifyBatch 对「开启 AI 分类且仍无领域标签」的文章分批补跑分类（跳过仍活跃的 pending）。
// delayBetween 为每条之间的休眠，降低模型接口压力。
func (p *ArticleAIProcessor) BackfillClassifyBatch(limit int, delayBetween time.Duration) (success, failed int) {
	if p == nil || p.db == nil || p.ai == nil || limit <= 0 {
		return 0, 0
	}
	feedIDs, err := classifyBackfillFeedIDs(p.db)
	if err != nil {
		logger.Error("ai backfill classify feed query: %v", err)
		return 0, 0
	}
	if len(feedIDs) == 0 {
		return 0, 0
	}
	stalePendingCutoff := time.Now().Add(-aiProcessPendingMaxAge)
	var arts []models.Article
	err = applyBackfillStalePendingFilter(
		p.db.Model(&models.Article{}).
			Select(articleBackfillClassifySelect).
			Preload("Feed").
			Where("feed_id IN ?", feedIDs).
			Where("(ai_category = '' OR ai_category IS NULL)"),
		stalePendingCutoff,
	).
		Order("created_at ASC").
		Limit(limit).
		Find(&arts).Error
	if err != nil {
		logger.Error("ai backfill classify query: %v", err)
		return 0, 0
	}
	for i := range arts {
		art := &arts[i]
		f := art.Feed
		if f.ID == 0 || f.AIModelID == nil || *f.AIModelID == 0 {
			continue
		}
		if !f.AIClassifyEnabled {
			continue
		}
		title := art.Title
		body := classifyBodyForAI(art.Content)
		needsTranslation := !articleHasCompletedTranslation(*art)
		if f.AITranslateEnabled && strings.TrimSpace(f.AITargetLanguage) != "" && needsTranslation {
			bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
			p.runClassifyAndTranslate(f.UserID, *f.AIModelID, &f, art.ID, title, bodyPlain, bodyHTML)
		} else {
			p.runClassifyOnly(f.UserID, *f.AIModelID, art.ID, title, body)
		}
		var after models.Article
		_ = p.db.First(&after, art.ID).Error
		if after.AIProcessStatus == models.AIProcessFailed {
			failed++
		} else if after.AIProcessStatus == models.AIProcessDone && strings.TrimSpace(after.AICategory) != "" {
			success++
		} else if after.AIProcessStatus == models.AIProcessDone {
			// 模型返回空 category，仍计为失败以便后续可再试（或人工处理）
			failed++
		}
		if i < len(arts)-1 && delayBetween > 0 {
			time.Sleep(delayBetween)
		}
	}
	if len(arts) > 0 {
		logger.Info("ai backfill classify: candidates=%d ok=%d fail=%d", len(arts), success, failed)
	}
	return success, failed
}

// BackfillTranslateBatch 对「开启 AI 翻译且仍无可用正文译文」的文章补跑翻译。
// 若订阅同时开启分类，则仅处理已有 ai_category 的条目（与异步流水线一致）。
func (p *ArticleAIProcessor) BackfillTranslateBatch(limit int, delayBetween time.Duration) (success, failed int) {
	if p == nil || p.db == nil || p.ai == nil || limit <= 0 {
		return 0, 0
	}
	feedIDs, classifyFeedIDs, err := translateBackfillFeedIDs(p.db)
	if err != nil {
		logger.Error("ai backfill translate feed query: %v", err)
		return 0, 0
	}
	if len(feedIDs) == 0 {
		return 0, 0
	}
	stalePendingCutoff := time.Now().Add(-aiProcessPendingMaxAge)
	var arts []models.Article
	err = applyBackfillStalePendingFilter(
		applyTranslateBackfillArticleFilters(
			p.db.Model(&models.Article{}).
				Select(articleBackfillTranslateSelect).
				Preload("Feed").
				Where("feed_id IN ?", feedIDs),
			classifyFeedIDs,
		),
		stalePendingCutoff,
	).
		Order("created_at ASC").
		Limit(limit).
		Find(&arts).Error
	if err != nil {
		logger.Error("ai backfill translate query: %v", err)
		return 0, 0
	}
	for i := range arts {
		art := &arts[i]
		f := art.Feed
		if f.ID == 0 || f.AIModelID == nil || *f.AIModelID == 0 {
			continue
		}
		if !f.AITranslateEnabled || strings.TrimSpace(f.AITargetLanguage) == "" {
			continue
		}
		title := art.Title
		bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
		mid := *f.AIModelID
		if f.AIClassifyEnabled && f.AITranslateEnabled {
			cat := strings.TrimSpace(art.AICategory)
			p.translateWithOptionalCategory(f.UserID, mid, &f, art.ID, title, bodyPlain, bodyHTML, cat)
		} else {
			p.runTranslateOnly(f.UserID, mid, &f, art.ID, title, bodyPlain, bodyHTML)
		}
		var after models.Article
		_ = p.db.First(&after, art.ID).Error
		if after.AIProcessStatus == models.AIProcessFailed {
			failed++
		} else if after.AIProcessStatus == models.AIProcessDone && articleHasCompletedTranslation(after) {
			success++
		} else if after.AIProcessStatus == models.AIProcessDone {
			failed++
		}
		if i < len(arts)-1 && delayBetween > 0 {
			time.Sleep(delayBetween)
		}
	}
	if len(arts) > 0 {
		logger.Info("ai backfill translate: candidates=%d ok=%d fail=%d", len(arts), success, failed)
	}
	return success, failed
}
