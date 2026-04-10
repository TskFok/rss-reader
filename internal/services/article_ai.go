package services

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ushopal/rss-reader/internal/models"
	"gorm.io/gorm"
)

var articleAIHTMLTagRe = regexp.MustCompile(`<[^>]*>`)

// ArticleAIProcessor 在新文章入库后异步调用 AI 做分类/翻译
type ArticleAIProcessor struct {
	db *gorm.DB
	ai *AIModelService
}

func NewArticleAIProcessor(db *gorm.DB, ai *AIModelService) *ArticleAIProcessor {
	return &ArticleAIProcessor{db: db, ai: ai}
}

// FeedNeedsAIProcessing 订阅是否应在入库后异步跑 AI
func FeedNeedsAIProcessing(f *models.Feed) bool {
	if f == nil {
		return false
	}
	if !f.AIClassifyEnabled && !f.AITranslateEnabled {
		return false
	}
	if f.AIModelID == nil || *f.AIModelID == 0 {
		return false
	}
	if f.AITranslateEnabled && strings.TrimSpace(f.AITargetLanguage) == "" {
		return false
	}
	return true
}

// Enqueue 异步处理（仅当订阅开启 AI 且模型有效时）
func (p *ArticleAIProcessor) Enqueue(feed *models.Feed, articleID uint) {
	if p == nil || feed == nil || !FeedNeedsAIProcessing(feed) {
		return
	}
	fd := *feed
	uid := fd.UserID
	aid := articleID
	go p.run(uid, fd, aid)
}

func stripMarkdownJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

func extractJSONObject(s string) string {
	s = stripMarkdownJSONFence(s)
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i >= 0 && j > i {
		return s[i : j+1]
	}
	return s
}

type aiArticleJSON struct {
	Category           string `json:"category"`
	CategoryTranslated string `json:"category_translated"`
	TitleTranslated    string `json:"title_translated"`
	ContentTranslated  string `json:"content_translated"`
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}

func plainFromHTMLForAI(s string) string {
	s = articleAIHTMLTagRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func (p *ArticleAIProcessor) applyAIFailed(articleID uint, errMsg string) {
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_process_status": models.AIProcessFailed,
		"ai_last_error":     truncateRunes(errMsg, 500),
	}).Error
}

func buildClassifySystemPrompt() string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader. Reply with a single JSON object only, no markdown code fences.\n")
	b.WriteString("Field: \"category\" only — short topic label (<= 80 chars) in Chinese.\n")
	return b.String()
}

func buildTranslateSystemPrompt(targetLang string, withCategoryHint bool) string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader. Reply with a single JSON object only, no markdown code fences.\n")
	if withCategoryHint {
		b.WriteString("Fields: \"category_translated\" (same meaning as the topic label, in ")
		b.WriteString(targetLang)
		b.WriteString("), \"title_translated\", \"content_translated\" (plain text, not HTML; translate title and body to ")
		b.WriteString(targetLang)
		b.WriteString(").\n")
	} else {
		b.WriteString("Fields: \"title_translated\", \"content_translated\" only — plain text translation to ")
		b.WriteString(targetLang)
		b.WriteString(".\n")
	}
	return b.String()
}

func (p *ArticleAIProcessor) run(userID uint, feed models.Feed, articleID uint) {
	if p.ai == nil || p.db == nil {
		return
	}
	var art models.Article
	if err := p.db.First(&art, articleID).Error; err != nil {
		return
	}
	if art.FeedID != feed.ID {
		return
	}
	var f models.Feed
	if err := p.db.First(&f, feed.ID).Error; err != nil {
		return
	}
	if !FeedNeedsAIProcessing(&f) {
		_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
			"ai_process_status": "",
			"ai_last_error":     "",
		}).Error
		return
	}
	modelID := *f.AIModelID

	title := art.Title
	body := plainFromHTMLForAI(art.Content)
	body = truncateRunes(body, 12000)

	// 同时开启分类+翻译：分两次模型调用。分类先落库，再异步完成翻译（仍在 Enqueue 的 goroutine 内，不阻塞 RSS 入库 Create）。
	if f.AIClassifyEnabled && f.AITranslateEnabled {
		p.runClassifyThenTranslate(userID, modelID, &f, articleID, title, body)
		return
	}

	if f.AIClassifyEnabled {
		p.runClassifyOnly(userID, modelID, articleID, title, body)
		return
	}

	p.runTranslateOnly(userID, modelID, &f, articleID, title, body)
}

func (p *ArticleAIProcessor) runClassifyOnly(userID uint, modelID uint, articleID uint, title, body string) {
	msgs := []chatMessage{
		{Role: "system", Content: buildClassifySystemPrompt()},
		{Role: "user", Content: "Title:\n" + title + "\n\nBody:\n" + body},
	}
	raw, err := p.ai.ChatCompletionText(userID, modelID, 4096, msgs)
	if err != nil {
		p.applyAIFailed(articleID, err.Error())
		return
	}
	payload := extractJSONObject(raw)
	var parsed aiArticleJSON
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		p.applyAIFailed(articleID, "解析模型 JSON 失败")
		return
	}
	updates := map[string]interface{}{
		"ai_process_status":      models.AIProcessDone,
		"ai_last_error":          "",
		"ai_category":            truncateRunes(strings.TrimSpace(parsed.Category), 250),
		"ai_category_translated": "",
		"title_translated":       "",
		"content_translated":     "",
	}
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(updates).Error
}

func (p *ArticleAIProcessor) runTranslateOnly(userID uint, modelID uint, f *models.Feed, articleID uint, title, body string) {
	target := f.AITargetLanguage
	msgs := []chatMessage{
		{Role: "system", Content: buildTranslateSystemPrompt(target, false)},
		{Role: "user", Content: "Title:\n" + title + "\n\nBody:\n" + body},
	}
	raw, err := p.ai.ChatCompletionText(userID, modelID, 8192, msgs)
	if err != nil {
		p.applyAIFailed(articleID, err.Error())
		return
	}
	payload := extractJSONObject(raw)
	var parsed aiArticleJSON
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		p.applyAIFailed(articleID, "解析模型 JSON 失败")
		return
	}
	updates := map[string]interface{}{
		"ai_process_status":      models.AIProcessDone,
		"ai_last_error":          "",
		"ai_category":            "",
		"ai_category_translated": "",
		"title_translated":       truncateRunes(strings.TrimSpace(parsed.TitleTranslated), 1000),
		"content_translated":     truncateRunes(strings.TrimSpace(parsed.ContentTranslated), 200000),
	}
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(updates).Error
}

func (p *ArticleAIProcessor) runClassifyThenTranslate(userID uint, modelID uint, f *models.Feed, articleID uint, title, body string) {
	msgsClassify := []chatMessage{
		{Role: "system", Content: buildClassifySystemPrompt()},
		{Role: "user", Content: "Title:\n" + title + "\n\nBody:\n" + body},
	}
	raw, err := p.ai.ChatCompletionText(userID, modelID, 4096, msgsClassify)
	if err != nil {
		p.applyAIFailed(articleID, err.Error())
		return
	}
	payload := extractJSONObject(raw)
	var parsedClass aiArticleJSON
	if err := json.Unmarshal([]byte(payload), &parsedClass); err != nil {
		p.applyAIFailed(articleID, "解析模型 JSON 失败")
		return
	}
	cat := truncateRunes(strings.TrimSpace(parsedClass.Category), 250)
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_category":            cat,
		"ai_category_translated": "",
		"title_translated":       "",
		"content_translated":     "",
	}).Error

	if err := p.db.First(f, f.ID).Error; err != nil {
		return
	}
	if !f.AITranslateEnabled || strings.TrimSpace(f.AITargetLanguage) == "" {
		_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
			"ai_process_status": models.AIProcessDone,
			"ai_last_error":     "",
		}).Error
		return
	}

	var art models.Article
	if err := p.db.First(&art, articleID).Error; err != nil {
		return
	}
	topicLine := strings.TrimSpace(art.AICategory)
	userTr := "Title:\n" + title + "\n\nBody:\n" + body
	if topicLine != "" {
		userTr = "Topic label:\n" + topicLine + "\n\n" + userTr
	}
	msgsTr := []chatMessage{
		{Role: "system", Content: buildTranslateSystemPrompt(f.AITargetLanguage, true)},
		{Role: "user", Content: userTr},
	}
	rawTr, err := p.ai.ChatCompletionText(userID, modelID, 8192, msgsTr)
	if err != nil {
		p.applyAIFailed(articleID, "翻译失败: "+err.Error())
		return
	}
	payloadTr := extractJSONObject(rawTr)
	var parsedTr aiArticleJSON
	if err := json.Unmarshal([]byte(payloadTr), &parsedTr); err != nil {
		p.applyAIFailed(articleID, "翻译解析 JSON 失败")
		return
	}
	updates := map[string]interface{}{
		"ai_process_status":      models.AIProcessDone,
		"ai_last_error":          "",
		"ai_category_translated": truncateRunes(strings.TrimSpace(parsedTr.CategoryTranslated), 250),
		"title_translated":       truncateRunes(strings.TrimSpace(parsedTr.TitleTranslated), 1000),
		"content_translated":     truncateRunes(strings.TrimSpace(parsedTr.ContentTranslated), 200000),
	}
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(updates).Error
}
