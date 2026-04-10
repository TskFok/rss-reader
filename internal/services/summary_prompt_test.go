package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSummaryUserContent_DefaultPrefix(t *testing.T) {
	arts := []ArticleForSummary{{Title: "T", Content: "C", FeedTitle: "F", PublishedAt: "2025-01-01"}}
	s := BuildSummaryUserContent(nil, arts)
	assert.True(t, strings.Contains(s, DefaultSummaryUserPrefix()[:40]))
	assert.Contains(t, s, "【F】")
	assert.Contains(t, s, "T")
}

func TestBuildSummaryChatMessages_WithSystem(t *testing.T) {
	arts := []ArticleForSummary{{Title: "T", Content: "C", FeedTitle: "F", PublishedAt: "2025-01-01"}}
	msgs := BuildSummaryChatMessages(&SummaryPromptOptions{SystemPrompt: "SYS", UserPrefix: "说人话。\n---"}, arts)
	require.Len(t, msgs, 2)
	assert.Equal(t, "system", msgs[0].Role)
	assert.Equal(t, "SYS", msgs[0].Content)
	assert.Equal(t, "user", msgs[1].Role)
	assert.Contains(t, msgs[1].Content, "说人话")
}
