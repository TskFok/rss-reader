package services

import (
	"strings"
	"time"

	"github.com/ushopal/rss-reader/internal/logger"
	"github.com/ushopal/rss-reader/internal/models"
)

// BackfillClassifyBatch 对「开启 AI 分类且仍无领域标签」的文章分批补跑分类（跳过 pending）。
// delayBetween 为每条之间的休眠，降低模型接口压力。
func (p *ArticleAIProcessor) BackfillClassifyBatch(limit int, delayBetween time.Duration) (success, failed int) {
	if p == nil || p.db == nil || p.ai == nil || limit <= 0 {
		return 0, 0
	}
	var arts []models.Article
	err := p.db.Model(&models.Article{}).
		Preload("Feed").
		Joins("INNER JOIN feeds ON feeds.id = articles.feed_id AND feeds.deleted_at IS NULL").
		Where("feeds.ai_classify_enabled = ?", true).
		Where("feeds.ai_model_id IS NOT NULL AND feeds.ai_model_id > ?", 0).
		Where("LENGTH(TRIM(COALESCE(articles.ai_category, ''))) = 0").
		Where("COALESCE(articles.ai_process_status, '') <> ?", models.AIProcessPending).
		Order("articles.created_at ASC").
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
		body := plainFromHTMLForAI(art.Content)
		body = truncateRunes(body, 12000)
		p.runClassifyOnly(f.UserID, *f.AIModelID, art.ID, title, body)
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

// BackfillTranslateBatch 对「开启 AI 翻译且仍无译文」的文章补跑翻译。
// 若订阅同时开启分类，则仅处理已有 ai_category 的条目（与异步流水线一致）。
func (p *ArticleAIProcessor) BackfillTranslateBatch(limit int, delayBetween time.Duration) (success, failed int) {
	if p == nil || p.db == nil || p.ai == nil || limit <= 0 {
		return 0, 0
	}
	var arts []models.Article
	err := p.db.Model(&models.Article{}).
		Preload("Feed").
		Joins("INNER JOIN feeds ON feeds.id = articles.feed_id AND feeds.deleted_at IS NULL").
		Where("feeds.ai_translate_enabled = ?", true).
		Where("LENGTH(TRIM(COALESCE(feeds.ai_target_language, ''))) > 0").
		Where("feeds.ai_model_id IS NOT NULL AND feeds.ai_model_id > ?", 0).
		Where("(LENGTH(TRIM(COALESCE(articles.title_translated, ''))) = 0 AND LENGTH(TRIM(COALESCE(articles.content_translated, ''))) = 0)").
		Where("(NOT feeds.ai_classify_enabled OR LENGTH(TRIM(COALESCE(articles.ai_category, ''))) > 0)").
		Where("COALESCE(articles.ai_process_status, '') <> ?", models.AIProcessPending).
		Order("articles.created_at ASC").
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
		bodyPlain := plainFromHTMLForAI(art.Content)
		bodyPlain = truncateRunes(bodyPlain, 12000)
		bodyHTML := truncateRunes(strings.TrimSpace(art.Content), 20000)
		if bodyHTML == "" {
			bodyHTML = bodyPlain
		}
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
		} else if after.AIProcessStatus == models.AIProcessDone &&
			(strings.TrimSpace(after.TitleTranslated) != "" || strings.TrimSpace(after.ContentTranslated) != "") {
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
