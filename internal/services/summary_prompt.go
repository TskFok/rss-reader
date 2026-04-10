package services

import "strings"

// SummaryPromptOptions 总结时可选的 Prompt；nil 或全空字段表示使用内置默认用户说明（仍拼接文章列表）
type SummaryPromptOptions struct {
	SystemPrompt string // 可选，作为独立 system 消息
	UserPrefix   string // 可选；为空则使用 DefaultSummaryUserPrefix()
}

// DefaultSummaryUserPrefix 内置：接在文章列表前的用户说明（其后为「---」与文章块）
func DefaultSummaryUserPrefix() string {
	return "以下是用户在指定时间范围内订阅的 RSS 文章列表，请用中文对这些内容进行概括性总结，提炼主要话题、重要信息与趋势。要求：\n1. 总结必须使用中文；\n2. 按主题或订阅源分组归纳；\n3. 突出重要新闻或变化；\n4. 控制在 800 字以内。\n\n---\n\n"
}

func resolvedSummaryUserPrefix(o *SummaryPromptOptions) string {
	if o == nil || strings.TrimSpace(o.UserPrefix) == "" {
		return DefaultSummaryUserPrefix()
	}
	p := strings.TrimSpace(o.UserPrefix)
	if strings.Contains(p, "---") {
		if !strings.HasSuffix(p, "\n") {
			p += "\n"
		}
		return p + "\n"
	}
	return p + "\n\n---\n\n"
}

func summarySystemMessage(o *SummaryPromptOptions) string {
	if o == nil {
		return ""
	}
	return strings.TrimSpace(o.SystemPrompt)
}

// BuildSummaryUserContent 拼接「说明前缀 + 文章块」，用于单条 user 消息（无 system 时整条作为 user）
func BuildSummaryUserContent(opts *SummaryPromptOptions, articles []ArticleForSummary) string {
	prefix := resolvedSummaryUserPrefix(opts)
	var sb strings.Builder
	sb.WriteString(prefix)
	for i, a := range articles {
		sb.WriteString("【")
		sb.WriteString(a.FeedTitle)
		sb.WriteString("】")
		sb.WriteString(a.PublishedAt)
		sb.WriteString(" - ")
		sb.WriteString(a.Title)
		sb.WriteString("\n")
		sb.WriteString(a.Content)
		if i < len(articles)-1 {
			sb.WriteString("\n\n")
		}
	}
	s := sb.String()
	if len(s) > 100000 {
		s = s[:100000] + "\n\n...(内容已截断)"
	}
	return s
}

// BuildSummaryChatMessages 生成发给模型的 messages（可选 system + user）
func BuildSummaryChatMessages(opts *SummaryPromptOptions, articles []ArticleForSummary) []chatMessage {
	userContent := BuildSummaryUserContent(opts, articles)
	sys := summarySystemMessage(opts)
	if sys != "" {
		return []chatMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: userContent},
		}
	}
	return []chatMessage{{Role: "user", Content: userContent}}
}
