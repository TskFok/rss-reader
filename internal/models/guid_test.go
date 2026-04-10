package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArticleGUIDHash(t *testing.T) {
	assert.Equal(t, "", ArticleGUIDHash("   "))
	h := ArticleGUIDHash("hello")
	assert.Len(t, h, 64)
	assert.True(t, IsArticleGUIDHash(h))
	assert.Equal(t, h, ArticleGUIDHash("hello"))
	assert.NotEqual(t, ArticleGUIDHash("hello"), ArticleGUIDHash("hellox"))

	long := strings.Repeat("a", 5000)
	hl := ArticleGUIDHash(long)
	assert.Len(t, hl, 64)
	assert.True(t, IsArticleGUIDHash(hl))
}
