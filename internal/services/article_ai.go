package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/tskfok/rss-reader/internal/models"

	"gorm.io/gorm"
)

// 手动 AI 分类/翻译
var (
	ErrManualAINoModel           = errors.New("请先在订阅中选择 AI 模型")
	ErrManualAINoTargetLang      = errors.New("请先在订阅中设置翻译目标语言")
	ErrManualAIAlreadyClassified = errors.New("该文章已有 AI 分类")
	ErrManualAIAlreadyTranslated = errors.New("该文章已有译文")
	ErrManualAIPending           = errors.New("AI 正在处理中，请稍后再试")
	ErrManualAIInvalidTargetLang = errors.New("不支持的目标语言，请选择：中文、英语、法语、德语、阿拉伯语")
)

var articleAIHTMLTagRe = regexp.MustCompile(`<[^>]*>`)

const (
	translateStreamCategoryMarker           = "[[[CATEGORY]]]"
	translateStreamCategoryTranslatedMarker = "[[[CATEGORY_TRANSLATED]]]"
	translateStreamCategoryTranslatedTypo   = "[[[CATEGORY_TRANSALTED]]]"
	translateStreamTitleMarker              = "[[[TITLE]]]"
	translateStreamContentMarker            = "[[[CONTENT_HTML]]]"
	errEmptyTranslationContent              = "AI 返回空正文译文，请重试"
	aiProcessPendingMaxAge                  = 30 * time.Minute
	articleAIDefaultQueueSize               = 100
	articleAIDefaultWorkers                 = 1
	articleAIDefaultMinInterval             = 600 * time.Millisecond
)

// ArticleAIProcessor 在新文章入库后异步调用 AI 做分类/翻译
type ArticleAIProcessor struct {
	db              *gorm.DB
	ai              *AIModelService
	queue           chan articleAIJob
	stop            chan struct{}
	stopOnce        sync.Once
	workerWG        sync.WaitGroup
	throttleMu      sync.Mutex
	nextAutoJobTime time.Time
	autoMinInterval time.Duration
}

type articleAIJob struct {
	userID    uint
	feed      models.Feed
	articleID uint
}

func NewArticleAIProcessor(db *gorm.DB, ai *AIModelService) *ArticleAIProcessor {
	p := &ArticleAIProcessor{
		db:              db,
		ai:              ai,
		queue:           make(chan articleAIJob, articleAIDefaultQueueSize),
		stop:            make(chan struct{}),
		autoMinInterval: articleAIDefaultMinInterval,
	}
	p.startAutoWorkers(articleAIDefaultWorkers)
	return p
}

func (p *ArticleAIProcessor) startAutoWorkers(workers int) {
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		p.workerWG.Add(1)
		go p.autoWorker()
	}
}

func (p *ArticleAIProcessor) autoWorker() {
	defer p.workerWG.Done()
	for {
		select {
		case <-p.stop:
			return
		case job := <-p.queue:
			p.run(job.userID, job.feed, job.articleID)
		}
	}
}

func (p *ArticleAIProcessor) waitForAutoModelCall() bool {
	if p.autoMinInterval <= 0 {
		return true
	}
	p.throttleMu.Lock()
	now := time.Now()
	startAt := now
	if p.nextAutoJobTime.After(now) {
		startAt = p.nextAutoJobTime
	}
	p.nextAutoJobTime = startAt.Add(p.autoMinInterval)
	p.throttleMu.Unlock()

	if delay := time.Until(startAt); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			return true
		case <-p.stop:
			return false
		}
	}
	return true
}

func (p *ArticleAIProcessor) Stop() {
	if p == nil || p.stop == nil {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stop)
		p.workerWG.Wait()
	})
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
	p.enqueueOne(*feed, articleID)
}

// EnqueueBatch 批量入队；共享一次订阅校验与 feed 拷贝，减少重复开销。
func (p *ArticleAIProcessor) EnqueueBatch(feed *models.Feed, articleIDs []uint) {
	if p == nil || feed == nil || !FeedNeedsAIProcessing(feed) || len(articleIDs) == 0 {
		return
	}
	fd := *feed
	for _, articleID := range articleIDs {
		p.enqueueOne(fd, articleID)
	}
}

func (p *ArticleAIProcessor) enqueueOne(fd models.Feed, articleID uint) {
	job := articleAIJob{userID: fd.UserID, feed: fd, articleID: articleID}
	if p.queue == nil {
		go p.run(job.userID, job.feed, job.articleID)
		return
	}
	select {
	case <-p.stop:
		if p.db != nil {
			p.applyAIFailed(articleID, "AI 自动处理队列已停止，请稍后重试")
		}
	case p.queue <- job:
	default:
		if p.db != nil {
			p.applyAIFailed(articleID, "AI 自动处理队列已满，请稍后重试")
		}
	}
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

func classifyBodyForAI(content string) string {
	return truncateRunes(plainFromHTMLForAI(content), 12000)
}

func translateBodiesForAI(content string) (bodyPlain, bodyHTML string) {
	bodyPlain = plainFromHTMLForAI(content)
	bodyHTML = strings.TrimSpace(content)
	if bodyHTML == "" {
		bodyHTML = bodyPlain
	}
	return bodyPlain, bodyHTML
}

func articleHasCompletedTranslation(art models.Article) bool {
	if strings.TrimSpace(art.ContentTranslated) == "" {
		return false
	}
	return art.AIProcessStatus != models.AIProcessFailed && art.AIProcessStatus != models.AIProcessPending
}

func normalizeAITranslation(parsed aiArticleJSON) (titleTranslated, contentTranslated string, ok bool) {
	titleTranslated = truncateRunes(strings.TrimSpace(parsed.TitleTranslated), 1000)
	contentTranslated = truncateRunes(strings.TrimSpace(parsed.ContentTranslated), 200000)
	return titleTranslated, contentTranslated, contentTranslated != ""
}

func (p *ArticleAIProcessor) applyAIFailed(articleID uint, errMsg string) {
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_process_status": models.AIProcessFailed,
		"ai_last_error":     truncateRunes(errMsg, 500),
	}).Error
}

func (p *ArticleAIProcessor) markAIPending(articleID uint) error {
	return p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_process_status": models.AIProcessPending,
		"ai_last_error":     "",
	}).Error
}

func aiProcessPendingIsStale(art models.Article, now time.Time) bool {
	if art.AIProcessStatus != models.AIProcessPending {
		return false
	}
	ts := art.UpdatedAt
	if ts.IsZero() {
		ts = art.CreatedAt
	}
	if ts.IsZero() {
		return true
	}
	return !ts.After(now.Add(-aiProcessPendingMaxAge))
}

func (p *ArticleAIProcessor) failPendingIfUnfinished(articleID uint, errMsg string) {
	var art models.Article
	if err := p.db.Select("id", "ai_process_status").First(&art, articleID).Error; err != nil {
		return
	}
	if art.AIProcessStatus != models.AIProcessPending {
		return
	}
	p.applyAIFailed(articleID, errMsg)
}

func buildClassifySystemPrompt() string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader. Reply with a single JSON object only, no markdown code fences.\n")
	b.WriteString("Field: \"category\" only — a concise Chinese DOMAIN label that summarizes what the article is broadly about (inductive bucketing / 领域归纳), NOT a fine-grained headline or keyword list.\n")
	b.WriteString("Prefer 2–8 characters. Use common high-level domains when applicable, e.g. 财经、军事、科技、体育、社会、国际、健康、文化、娱乐、房产、汽车、教育、科学、环境、政治；")
	b.WriteString("pick ONE best-fitting label. Do not use full sentences, product names alone, or person names as the category.\n")
	b.WriteString("Max length 80 chars (rarely needed); usually a short domain word is enough.\n")
	return b.String()
}

func buildTranslateSystemPrompt(targetLang string, withCategoryHint bool) string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader. Reply with a single JSON object only, no markdown code fences.\n")
	if withCategoryHint {
		b.WriteString("Fields: \"category_translated\" (same meaning as the topic label, in ")
		b.WriteString(targetLang)
		b.WriteString("), \"title_translated\", \"content_translated\" (an HTML fragment translated to ")
		b.WriteString(targetLang)
		b.WriteString(").\n")
	} else {
		b.WriteString("Fields: \"title_translated\", \"content_translated\" only — translate to ")
		b.WriteString(targetLang)
		b.WriteString(", and return \"content_translated\" as an HTML fragment.\n")
	}
	b.WriteString("For \"content_translated\", preserve the original HTML structure, element order, links, image/audio/video/iframe/embed nodes, attributes, and non-text elements. Only translate human-readable text content inside the HTML. Do not drop elements. Do not convert HTML to Markdown or plain text.\n")
	return b.String()
}

func buildClassifyTranslateSystemPrompt(targetLang string) string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader. Reply with a single JSON object only, no markdown code fences.\n")
	b.WriteString("Fields: \"category\" (a concise Chinese DOMAIN label), \"category_translated\" (same meaning as category, in ")
	b.WriteString(targetLang)
	b.WriteString("), \"title_translated\", \"content_translated\" (an HTML fragment translated to ")
	b.WriteString(targetLang)
	b.WriteString(").\n")
	b.WriteString("For \"category\", summarize what the article is broadly about (inductive bucketing / 领域归纳), NOT a fine-grained headline or keyword list. Prefer 2–8 characters. Use common high-level domains when applicable, e.g. 财经、军事、科技、体育、社会、国际、健康、文化、娱乐、房产、汽车、教育、科学、环境、政治; pick ONE best-fitting label.\n")
	b.WriteString("For \"content_translated\", preserve the original HTML structure, element order, links, image/audio/video/iframe/embed nodes, attributes, and non-text elements. Only translate human-readable text content inside the HTML. Do not drop elements. Do not convert HTML to Markdown or plain text.\n")
	return b.String()
}

func buildTranslateStreamSystemPrompt(targetLang string) string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader.\n")
	b.WriteString("Translate the provided title and HTML body into ")
	b.WriteString(targetLang)
	b.WriteString(".\n")
	b.WriteString("For body translation, preserve the original HTML structure, element order, links, image/audio/video/iframe/embed nodes, attributes, and non-text elements. Only translate human-readable text content inside the HTML.\n")
	b.WriteString("Do not add explanations or markdown code fences.\n")
	b.WriteString("Return output in the exact format below:\n")
	b.WriteString(translateStreamTitleMarker + "\n")
	b.WriteString("<translated title>\n")
	b.WriteString(translateStreamContentMarker + "\n")
	b.WriteString("<translated html fragment>\n")
	return b.String()
}

func buildClassifyTranslateStreamSystemPrompt(targetLang string) string {
	var b strings.Builder
	b.WriteString("You are a helper for an RSS reader.\n")
	b.WriteString("Classify and translate the provided title and HTML body in one response.\n")
	b.WriteString("For category, return a concise Chinese DOMAIN label that summarizes what the article is broadly about (inductive bucketing / 领域归纳), NOT a fine-grained headline or keyword list. Prefer 2–8 characters; pick ONE best-fitting label.\n")
	b.WriteString("Translate the category, title, and HTML body into ")
	b.WriteString(targetLang)
	b.WriteString(".\n")
	b.WriteString("For body translation, preserve the original HTML structure, element order, links, image/audio/video/iframe/embed nodes, attributes, and non-text elements. Only translate human-readable text content inside the HTML.\n")
	b.WriteString("Do not add explanations or markdown code fences.\n")
	b.WriteString("Return output in the exact format below:\n")
	b.WriteString(translateStreamCategoryMarker + "\n")
	b.WriteString("<Chinese category>\n")
	b.WriteString(translateStreamCategoryTranslatedMarker + "\n")
	b.WriteString("<translated category>\n")
	b.WriteString(translateStreamTitleMarker + "\n")
	b.WriteString("<translated title>\n")
	b.WriteString(translateStreamContentMarker + "\n")
	b.WriteString("<translated html fragment>\n")
	b.WriteString("Use these markers exactly as written.\n")
	return b.String()
}

func (p *ArticleAIProcessor) run(userID uint, feed models.Feed, articleID uint) {
	if p.ai == nil || p.db == nil {
		return
	}
	claimed := false
	defer func() {
		if !claimed {
			return
		}
		if r := recover(); r != nil {
			p.failPendingIfUnfinished(articleID, fmt.Sprintf("AI 处理异常：%v", r))
			return
		}
		p.failPendingIfUnfinished(articleID, "AI 处理未完成，请重试")
	}()
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
	if err := p.markAIPending(articleID); err != nil {
		return
	}
	claimed = true
	modelID := *f.AIModelID

	title := art.Title

	// 同时开启分类+翻译：一次模型调用返回分类、分类译名、标题译文和正文译文（仍在 Enqueue 的 goroutine 内，不阻塞 RSS 入库 Create）。
	if f.AIClassifyEnabled && f.AITranslateEnabled {
		if !p.waitForAutoModelCall() {
			return
		}
		bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
		p.runClassifyAndTranslate(userID, modelID, &f, articleID, title, bodyPlain, bodyHTML)
		return
	}

	if f.AIClassifyEnabled {
		if !p.waitForAutoModelCall() {
			return
		}
		bodyPlain := classifyBodyForAI(art.Content)
		p.runClassifyOnly(userID, modelID, articleID, title, bodyPlain)
		return
	}

	if !p.waitForAutoModelCall() {
		return
	}
	bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
	p.runTranslateOnly(userID, modelID, &f, articleID, title, bodyPlain, bodyHTML)
}

func (p *ArticleAIProcessor) translateWithOptionalCategory(userID uint, modelID uint, f *models.Feed, articleID uint, title, bodyPlain, bodyHTML, categoryLine string) {
	target := f.AITargetLanguage
	topic := strings.TrimSpace(categoryLine)
	withHint := topic != ""
	msgs := []chatMessage{
		{Role: "system", Content: buildTranslateSystemPrompt(target, withHint)},
		{Role: "user", Content: buildTranslateUserContent(topic, title, bodyHTML, bodyPlain)},
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
	titleTranslated, contentTranslated, ok := normalizeAITranslation(parsed)
	if !ok {
		p.applyAIFailed(articleID, errEmptyTranslationContent)
		return
	}
	updates := map[string]interface{}{
		"ai_process_status":  models.AIProcessDone,
		"ai_last_error":      "",
		"title_translated":   titleTranslated,
		"content_translated": contentTranslated,
	}
	if withHint {
		updates["ai_category_translated"] = truncateRunes(strings.TrimSpace(parsed.CategoryTranslated), 250)
	} else {
		updates["ai_category"] = ""
		updates["ai_category_translated"] = ""
	}
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(updates).Error
}

func buildTranslateUserContent(topicLine, title, bodyHTML, bodyPlain string) string {
	userTr := "Title:\n" + title + "\n\nBody HTML:\n" + bodyHTML
	if strings.TrimSpace(bodyPlain) != "" && bodyPlain != bodyHTML {
		userTr += "\n\nBody plain text (for reference):\n" + bodyPlain
	}
	if strings.TrimSpace(topicLine) != "" {
		userTr = "Topic label:\n" + strings.TrimSpace(topicLine) + "\n\n" + userTr
	}
	return userTr
}

func buildTranslateStreamUserContent(topicLine, title, bodyHTML, bodyPlain string) string {
	return buildTranslateUserContent(topicLine, title, bodyHTML, bodyPlain)
}

func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	if j := strings.LastIndex(s, "```"); j >= 0 {
		s = s[:j]
	}
	return strings.TrimSpace(s)
}

func parseTranslateStreamOutput(raw string) (titleTranslated string, contentTranslated string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	contentIdx := strings.Index(raw, translateStreamContentMarker)
	if contentIdx < 0 {
		return "", stripMarkdownFence(raw)
	}
	titleIdx := strings.Index(raw, translateStreamTitleMarker)
	if titleIdx >= 0 && titleIdx < contentIdx {
		titleTranslated = strings.TrimSpace(raw[titleIdx+len(translateStreamTitleMarker) : contentIdx])
		titleTranslated = stripMarkdownFence(titleTranslated)
	}
	contentTranslated = strings.TrimSpace(raw[contentIdx+len(translateStreamContentMarker):])
	contentTranslated = stripMarkdownFence(contentTranslated)
	return strings.TrimSpace(titleTranslated), strings.TrimSpace(contentTranslated)
}

func firstMarkerIndex(s string, markers ...string) int {
	first := -1
	for _, marker := range markers {
		if marker == "" {
			continue
		}
		idx := strings.Index(s, marker)
		if idx < 0 {
			continue
		}
		if first < 0 || idx < first {
			first = idx
		}
	}
	return first
}

func streamTextBetweenAny(raw string, startMarkers []string, endMarkers ...string) string {
	startIdx := -1
	startLen := 0
	for _, marker := range startMarkers {
		idx := strings.Index(raw, marker)
		if idx < 0 {
			continue
		}
		if startIdx < 0 || idx < startIdx {
			startIdx = idx
			startLen = len(marker)
		}
	}
	if startIdx < 0 {
		return ""
	}
	out := raw[startIdx+startLen:]
	if endIdx := firstMarkerIndex(out, endMarkers...); endIdx >= 0 {
		out = out[:endIdx]
	}
	return strings.TrimSpace(stripMarkdownFence(out))
}

func parseClassifyTranslateStreamOutput(raw string) (category, categoryTranslated, titleTranslated, contentTranslated string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", ""
	}
	category = streamTextBetweenAny(
		raw,
		[]string{translateStreamCategoryMarker},
		translateStreamCategoryTranslatedMarker,
		translateStreamCategoryTranslatedTypo,
		translateStreamTitleMarker,
		translateStreamContentMarker,
	)
	categoryTranslated = streamTextBetweenAny(
		raw,
		[]string{translateStreamCategoryTranslatedMarker, translateStreamCategoryTranslatedTypo},
		translateStreamTitleMarker,
		translateStreamContentMarker,
	)
	titleTranslated = streamTextBetweenAny(raw, []string{translateStreamTitleMarker}, translateStreamContentMarker)
	contentTranslated = streamTextBetweenAny(raw, []string{translateStreamContentMarker})
	return category, categoryTranslated, titleTranslated, contentTranslated
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

func (p *ArticleAIProcessor) runTranslateOnly(userID uint, modelID uint, f *models.Feed, articleID uint, title, bodyPlain, bodyHTML string) {
	p.translateWithOptionalCategory(userID, modelID, f, articleID, title, bodyPlain, bodyHTML, "")
}

func (p *ArticleAIProcessor) runClassifyAndTranslate(userID uint, modelID uint, f *models.Feed, articleID uint, title, bodyPlain, bodyHTML string) {
	msgs := []chatMessage{
		{Role: "system", Content: buildClassifyTranslateSystemPrompt(f.AITargetLanguage)},
		{Role: "user", Content: buildTranslateUserContent("", title, bodyHTML, bodyPlain)},
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
	titleTranslated, contentTranslated, ok := normalizeAITranslation(parsed)
	if !ok {
		p.applyAIFailed(articleID, errEmptyTranslationContent)
		return
	}
	_ = p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_process_status":      models.AIProcessDone,
		"ai_last_error":          "",
		"ai_category":            truncateRunes(strings.TrimSpace(parsed.Category), 250),
		"ai_category_translated": truncateRunes(strings.TrimSpace(parsed.CategoryTranslated), 250),
		"title_translated":       titleTranslated,
		"content_translated":     contentTranslated,
	}).Error
}

// ManualClassify 同步手动分类（文章尚无分类）。overrideModelID 非空时使用该模型（须属于当前用户），否则使用订阅默认模型。
func (p *ArticleAIProcessor) ManualClassify(userID uint, articleID uint, overrideModelID *uint) error {
	if p == nil || p.db == nil || p.ai == nil {
		return errors.New("AI 服务不可用")
	}
	var art models.Article
	if err := p.db.Preload("Feed").First(&art, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	f := art.Feed
	if f.UserID != userID {
		return ErrArticleNotFound
	}
	if art.AIProcessStatus == models.AIProcessPending && !aiProcessPendingIsStale(art, time.Now()) {
		return ErrManualAIPending
	}
	if strings.TrimSpace(art.AICategory) != "" {
		return ErrManualAIAlreadyClassified
	}
	modelID, err := p.resolveManualModelID(userID, &f, overrideModelID)
	if err != nil {
		return err
	}
	title := art.Title
	body := classifyBodyForAI(art.Content)
	p.runClassifyOnly(userID, modelID, articleID, title, body)
	return p.errIfArticleAIFailed(articleID)
}

// ManualTranslate 同步手动翻译（尚无可用正文译文）；若尚无分类，则同一次模型调用生成分类与译文。
// overrideModelID / overrideTargetLang 可临时覆盖订阅；语言为空串时表示使用订阅默认目标语言。
func (p *ArticleAIProcessor) ManualTranslate(userID uint, articleID uint, overrideModelID *uint, overrideTargetLang string) error {
	if p == nil || p.db == nil || p.ai == nil {
		return errors.New("AI 服务不可用")
	}
	var art models.Article
	if err := p.db.Preload("Feed").First(&art, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	f := art.Feed
	if f.UserID != userID {
		return ErrArticleNotFound
	}
	if art.AIProcessStatus == models.AIProcessPending && !aiProcessPendingIsStale(art, time.Now()) {
		return ErrManualAIPending
	}
	if articleHasCompletedTranslation(art) {
		return ErrManualAIAlreadyTranslated
	}
	modelID, err := p.resolveManualModelID(userID, &f, overrideModelID)
	if err != nil {
		return err
	}
	ol := strings.TrimSpace(overrideTargetLang)
	var targetLang string
	if ol != "" {
		if !isAllowedManualTargetLang(ol) {
			return ErrManualAIInvalidTargetLang
		}
		targetLang = ol
	} else {
		targetLang = strings.TrimSpace(f.AITargetLanguage)
	}
	if targetLang == "" {
		return ErrManualAINoTargetLang
	}
	ff := f
	ff.AITargetLanguage = targetLang
	mid := modelID
	ff.AIModelID = &mid
	title := art.Title
	bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
	if strings.TrimSpace(art.AICategory) == "" {
		p.runClassifyAndTranslate(userID, modelID, &ff, articleID, title, bodyPlain, bodyHTML)
		return p.errIfArticleAIFailed(articleID)
	}
	p.translateWithOptionalCategory(userID, modelID, &ff, articleID, title, bodyPlain, bodyHTML, strings.TrimSpace(art.AICategory))
	return p.errIfArticleAIFailed(articleID)
}

// ManualTranslateStream 流式手动翻译（仅当文章尚无可用正文译文）；若尚无分类，则同一次模型调用生成分类与译文。
func (p *ArticleAIProcessor) ManualTranslateStream(userID uint, articleID uint, overrideModelID *uint, overrideTargetLang string, onChunk func(string) error) error {
	if p == nil || p.db == nil || p.ai == nil {
		return errors.New("AI 服务不可用")
	}
	var art models.Article
	if err := p.db.Preload("Feed").First(&art, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	f := art.Feed
	if f.UserID != userID {
		return ErrArticleNotFound
	}
	if art.AIProcessStatus == models.AIProcessPending && !aiProcessPendingIsStale(art, time.Now()) {
		return ErrManualAIPending
	}
	if articleHasCompletedTranslation(art) {
		return ErrManualAIAlreadyTranslated
	}
	modelID, err := p.resolveManualModelID(userID, &f, overrideModelID)
	if err != nil {
		return err
	}
	ol := strings.TrimSpace(overrideTargetLang)
	var targetLang string
	if ol != "" {
		if !isAllowedManualTargetLang(ol) {
			return ErrManualAIInvalidTargetLang
		}
		targetLang = ol
	} else {
		targetLang = strings.TrimSpace(f.AITargetLanguage)
	}
	if targetLang == "" {
		return ErrManualAINoTargetLang
	}
	title := art.Title
	bodyPlain, bodyHTML := translateBodiesForAI(art.Content)
	if err := p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"ai_process_status": models.AIProcessPending,
		"ai_last_error":     "",
	}).Error; err != nil {
		return err
	}

	topic := strings.TrimSpace(art.AICategory)
	shouldClassify := topic == ""
	msgs := []chatMessage{
		{Role: "system", Content: buildTranslateStreamSystemPrompt(targetLang)},
		{Role: "user", Content: buildTranslateStreamUserContent(topic, title, bodyHTML, bodyPlain)},
	}
	if shouldClassify {
		msgs = []chatMessage{
			{Role: "system", Content: buildClassifyTranslateStreamSystemPrompt(targetLang)},
			{Role: "user", Content: buildTranslateStreamUserContent("", title, bodyHTML, bodyPlain)},
		}
	}

	var raw strings.Builder
	contentStart := -1
	sentLen := 0
	err = p.ai.ChatCompletionStream(userID, modelID, 8192, msgs, func(delta string) error {
		raw.WriteString(delta)
		buf := raw.String()
		if contentStart < 0 {
			idx := strings.Index(buf, translateStreamContentMarker)
			if idx < 0 {
				return nil
			}
			contentStart = idx + len(translateStreamContentMarker)
		}
		if onChunk == nil {
			return nil
		}
		content := buf[contentStart:]
		if sentLen >= len(content) {
			return nil
		}
		out := content[sentLen:]
		sentLen = len(content)
		if out == "" {
			return nil
		}
		return onChunk(out)
	})
	if err != nil {
		p.applyAIFailed(articleID, err.Error())
		return err
	}

	var category, categoryTranslated, titleTranslated, contentTranslated string
	if shouldClassify {
		category, categoryTranslated, titleTranslated, contentTranslated = parseClassifyTranslateStreamOutput(raw.String())
	} else {
		titleTranslated, contentTranslated = parseTranslateStreamOutput(raw.String())
	}
	contentTranslated = truncateRunes(strings.TrimSpace(contentTranslated), 200000)
	if contentTranslated == "" {
		p.applyAIFailed(articleID, "解析模型流式输出失败")
		return errors.New("解析模型流式输出失败")
	}
	updates := map[string]interface{}{
		"ai_process_status":  models.AIProcessDone,
		"ai_last_error":      "",
		"title_translated":   truncateRunes(strings.TrimSpace(titleTranslated), 1000),
		"content_translated": contentTranslated,
	}
	if shouldClassify {
		updates["ai_category"] = truncateRunes(strings.TrimSpace(category), 250)
		updates["ai_category_translated"] = truncateRunes(strings.TrimSpace(categoryTranslated), 250)
	}
	if err := p.db.Model(&models.Article{}).Where("id = ?", articleID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func isAllowedManualTargetLang(code string) bool {
	switch strings.TrimSpace(code) {
	case "zh-CN", "en", "fr", "de", "ar":
		return true
	default:
		return false
	}
}

func (p *ArticleAIProcessor) resolveManualModelID(userID uint, f *models.Feed, override *uint) (uint, error) {
	if override != nil && *override != 0 {
		_, err := p.ai.GetByID(userID, *override)
		if err != nil {
			if errors.Is(err, ErrAIModelNotFound) {
				return 0, ErrAIModelNotFound
			}
			return 0, err
		}
		return *override, nil
	}
	if f.AIModelID != nil && *f.AIModelID != 0 {
		return *f.AIModelID, nil
	}
	return 0, ErrManualAINoModel
}

func (p *ArticleAIProcessor) errIfArticleAIFailed(articleID uint) error {
	var art models.Article
	if err := p.db.First(&art, articleID).Error; err != nil {
		return err
	}
	if art.AIProcessStatus == models.AIProcessFailed {
		msg := strings.TrimSpace(art.AILastError)
		if msg == "" {
			msg = "AI 处理失败"
		}
		return errors.New(msg)
	}
	return nil
}
