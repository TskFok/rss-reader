package services

import (
	"errors"
	"time"
)

// RunDailySummaryForYesterday 执行一次“昨天”的分页总结，直到某页文章数为 0。
// 每页生成一条总结历史记录。
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
				AIModelID:    aiModelID,
				FeedIDs:      feedIDs,
				StartTime:    startStr,
				EndTime:      endStr,
				Page:         page,
				PageSize:     pageSize,
				Order:        order,
				ArticleCount: 0,
				Total:        total,
				Content:      "",
				Error:        err.Error(),
			})
			return err
		}
		if len(items) == 0 {
			break
		}
		content, sumErr := aiModelSvc.Summarize(userID, aiModelID, items)
		errStr := ""
		if sumErr != nil {
			errStr = sumErr.Error()
		}
		// 保存历史
		_, _ = historySvc.Create(userID, CreateSummaryHistoryRequest{
			AIModelID:    aiModelID,
			FeedIDs:      feedIDs,
			StartTime:    startStr,
			EndTime:      endStr,
			Page:         page,
			PageSize:     pageSize,
			Order:        order,
			ArticleCount: len(items),
			Total:        total,
			Content:      content,
			Error:        errStr,
		})

		// 模型失败时不再继续翻页，避免连续失败刷屏
		if sumErr != nil {
			return sumErr
		}
		page++
		// 安全阈值：避免意外无限循环
		if page > 1000 {
			_, _ = historySvc.Create(userID, CreateSummaryHistoryRequest{
				AIModelID: aiModelID,
				FeedIDs:   feedIDs,
				StartTime: startStr,
				EndTime:   endStr,
				Page:      page,
				PageSize:  pageSize,
				Order:     order,
				Error:     "分页次数过多，已中止",
				Content:   "",
			})
			return errors.New("分页次数过多，已中止")
		}
	}
	return nil
}

