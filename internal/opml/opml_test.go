package opml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ushopal/rss-reader/internal/models"
)

func TestGenerateAndParse(t *testing.T) {
	feeds := []models.Feed{
		{
			URL:   "https://example.com/a",
			Title: "A",
			Category: &models.FeedCategory{
				Name: "科技",
			},
		},
		{
			URL:   "https://example.com/b",
			Title: "B",
		},
	}

	data, err := Generate("test", feeds)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "opml")
	assert.Contains(t, string(data), "科技")
	assert.Contains(t, string(data), "https://example.com/a")

	items, err := Parse(data)
	assert.NoError(t, err)
	assert.Len(t, items, 2)

	var catNames []string
	for _, it := range items {
		if it.URL == "https://example.com/a" {
			assert.Equal(t, "科技", it.Category)
		}
		if it.URL == "https://example.com/b" {
			assert.Equal(t, "", it.Category)
		}
		catNames = append(catNames, it.Category)
	}
}

