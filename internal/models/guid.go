package models

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// ArticleGUIDHash 将 RSS 条目标识（guid 或 link）规范为 64 位十六进制 SHA256，用作 articles.guid 列。
func ArticleGUIDHash(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

var articleGUIDHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// IsArticleGUIDHash 判断是否为 articles.guid 使用的 64 位小写十六进制形式。
func IsArticleGUIDHash(s string) bool {
	return articleGUIDHex64.MatchString(s)
}
