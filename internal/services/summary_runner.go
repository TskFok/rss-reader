package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ushopal/rss-reader/internal/logger"
	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

const feishuAlertErrMaxLen = 500

var summaryAutoRetryDelays = []time.Duration{
	time.Minute,
	3 * time.Minute,
	10 * time.Minute,
	60 * time.Minute,
}

var summaryAutoRetrySleep = time.Sleep

// RunDailySummaryForYesterday 执行一次“昨天”的分页总结，直到某页文章数为 0。
// 每页生成一条总结历史记录。
// feishuBot 为 nil 时不发送飞书告警；db 在 feishuBot 非 nil 时用于查询用户 Webhook 和模型信息。
func RunDailySummaryForYesterday(
	userID uint,
	aiModelSvc *AIModelService,
	articleSvc *ArticleService,
	historySvc *SummaryHistoryService,
	aiModelID uint,
	feedIDs []uint,
	pageSize int,
	order string,
	now time.Time,
	loc *time.Location,
	feishuBot FeishuBotClient,
	db *gorm.DB,
	prompt *SummaryPromptOptions,
	summaryTemplateID *uint,
	summaryTemplateName string,
) error {
	if loc != nil {
		now = now.In(loc)
	}
	// 昨天（按上海时区日期计算）
	yesterday := now.AddDate(0, 0, -1)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	end := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, yesterday.Location())
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	// 生成时仍复用现有查询逻辑：start/end 传入 time，handler 中对 end 会加 24h-1s，这里直接传整日范围
	page := 1
	for {
		items, total, err := articleSvc.ListForSummary(userID, feedIDs, &start, &end, page, pageSize, order)
		if err != nil {
			_, _ = historySvc.Create(userID, CreateSummaryHistoryRequest{
				AIModelID:           aiModelID,
				SummaryTemplateID:   summaryTemplateID,
				SummaryTemplateName:   summaryTemplateName,
				FeedIDs:               feedIDs,
				StartTime:             startStr,
				EndTime:               endStr,
				Page:                  page,
				PageSize:              pageSize,
				Order:                 order,
				ArticleCount:          0,
				Total:                 total,
				Content:               "",
				Error:                 err.Error(),
			})
			trySendFeishuAlert(feishuBot, db, userID, aiModelID, startStr, endStr, page, pageSize, order, 0, err.Error())
			return err
		}
		if len(items) == 0 {
			break
		}
		content, sumErr := summarizeWithAutoRetry(func() (string, error) {
			return aiModelSvc.Summarize(userID, aiModelID, items, prompt)
		})
		errStr := ""
		if sumErr != nil {
			errStr = sumErr.Error()
		}
		// 保存历史
		_, _ = historySvc.Create(userID, CreateSummaryHistoryRequest{
			AIModelID:           aiModelID,
			SummaryTemplateID:   summaryTemplateID,
			SummaryTemplateName:   summaryTemplateName,
			FeedIDs:               feedIDs,
			StartTime:             startStr,
			EndTime:               endStr,
			Page:                  page,
			PageSize:              pageSize,
			Order:                 order,
			ArticleCount:          len(items),
			Total:                 total,
			Content:               content,
			Error:                 errStr,
		})

		// 模型失败时：记录错误并继续下一页，避免阻塞整次总结
		if sumErr != nil {
			trySendFeishuAlert(feishuBot, db, userID, aiModelID, startStr, endStr, page, pageSize, order, len(items), sumErr.Error())
			page++
			continue
		}
		page++
		// 安全阈值：避免意外无限循环
		if page > 1000 {
			errMsg := "分页次数过多，已中止"
			_, _ = historySvc.Create(userID, CreateSummaryHistoryRequest{
				AIModelID:           aiModelID,
				SummaryTemplateID:   summaryTemplateID,
				SummaryTemplateName: summaryTemplateName,
				FeedIDs:             feedIDs,
				StartTime:           startStr,
				EndTime:             endStr,
				Page:                page,
				PageSize:            pageSize,
				Order:               order,
				Error:               errMsg,
				Content:             "",
			})
			trySendFeishuAlert(feishuBot, db, userID, aiModelID, startStr, endStr, page, pageSize, order, 0, errMsg)
			return errors.New(errMsg)
		}
	}
	return nil
}

func summarizeWithAutoRetry(runOnce func() (string, error)) (string, error) {
	content, err := runOnce()
	if err == nil {
		return content, nil
	}
	for _, delay := range summaryAutoRetryDelays {
		if delay > 0 {
			summaryAutoRetrySleep(delay)
		}
		content, err = runOnce()
		if err == nil {
			return content, nil
		}
	}
	return content, err
}

// trySendFeishuAlert 在定时总结失败时尝试发送飞书告警，仅当用户配置了 Webhook 时发送。
// 发送失败不影响主流程，仅记录日志。
func trySendFeishuAlert(
	feishuBot FeishuBotClient,
	db *gorm.DB,
	userID uint,
	aiModelID uint,
	startStr, endStr string,
	page, pageSize int,
	order string,
	articleCount int,
	errMsg string,
) {
	if feishuBot == nil || db == nil {
		return
	}
	var user models.User
	if err := db.Select("username", "feishu_bot_webhook", "feishu_notify_type", "feishu_id").
		Where("id = ?", userID).First(&user).Error; err != nil {
		return
	}
	notifyType := strings.TrimSpace(user.FeishuNotifyType)
	webhook := strings.TrimSpace(user.FeishuBotWebhook)
	if notifyType == "" && webhook != "" {
		notifyType = "webhook"
	}
	if notifyType == "" {
		return
	}
	modelName := ""
	var model models.AIModel
	if err := db.Select("name").Where("user_id = ? AND id = ?", userID, aiModelID).First(&model).Error; err == nil {
		modelName = model.Name
	}
	if modelName == "" {
		modelName = "(未知)"
	}
	truncated := truncateString(errMsg, feishuAlertErrMaxLen)
	title := "[RSS Reader 定时总结失败告警]"
	content := strings.Join([]string{
		"用户：" + user.Username + " (ID: " + strconv.FormatUint(uint64(userID), 10) + ")",
		"模型：" + modelName + " (ID: " + strconv.FormatUint(uint64(aiModelID), 10) + ")",
		"时间范围：" + startStr + " ~ " + endStr,
		"页码：" + fmt.Sprintf("%d", page) + " / page_size=" + fmt.Sprintf("%d", pageSize) + ", order=" + order,
		"文章数：" + fmt.Sprintf("%d", articleCount),
		"错误：" + truncated,
	}, "\n")
	var sendErr error
	if notifyType == "webhook" && webhook != "" {
		sendErr = feishuBot.SendText(webhook, title, content)
	} else if notifyType == "api" && user.FeishuID != nil && strings.TrimSpace(*user.FeishuID) != "" {
		sendErr = feishuBot.SendToUserByOpenID(strings.TrimSpace(*user.FeishuID), title, content)
	}
	if sendErr != nil {
		logger.Warn("飞书告警发送失败 (user=%d): %v", userID, sendErr)
	}
}

func truncateString(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "..."
}


