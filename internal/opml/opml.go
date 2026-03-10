package opml

import (
	"encoding/xml"
	"sort"

	"github.com/ushopal/rss-reader/internal/models"
)

// Item 表示从 OPML 中解析出的订阅项
type Item struct {
	Category string
	URL      string
	Title    string
}

type document struct {
	XMLName xml.Name  `xml:"opml"`
	Version string    `xml:"version,attr"`
	Head    head      `xml:"head"`
	Body    body      `xml:"body"`
}

type head struct {
	Title string `xml:"title"`
}

type body struct {
	Outlines []outline `xml:"outline"`
}

type outline struct {
	Text     string    `xml:"text,attr,omitempty"`
	Title    string    `xml:"title,attr,omitempty"`
	Type     string    `xml:"type,attr,omitempty"`
	XMLURL   string    `xml:"xmlUrl,attr,omitempty"`
	Outlines []outline `xml:"outline"`
}

// Generate 依据用户订阅生成 OPML 文本
func Generate(title string, feeds []models.Feed) ([]byte, error) {
	byCategory := make(map[string][]models.Feed)
	var uncategorized []models.Feed
	for _, f := range feeds {
		if f.Category != nil && f.Category.Name != "" {
			byCategory[f.Category.Name] = append(byCategory[f.Category.Name], f)
		} else {
			uncategorized = append(uncategorized, f)
		}
	}

	var catNames []string
	for name := range byCategory {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	var top []outline

	for _, name := range catNames {
		fs := byCategory[name]
		sort.Slice(fs, func(i, j int) bool {
			return (fs[i].Title + fs[i].URL) < (fs[j].Title + fs[j].URL)
		})
		o := outline{
			Text:  name,
			Title: name,
		}
		for _, f := range fs {
			o.Outlines = append(o.Outlines, outline{
				Text:   firstNonEmpty(f.Title, f.URL),
				Title:  f.Title,
				Type:   "rss",
				XMLURL: f.URL,
			})
		}
		top = append(top, o)
	}

	if len(uncategorized) > 0 {
		sort.Slice(uncategorized, func(i, j int) bool {
			return (uncategorized[i].Title + uncategorized[i].URL) < (uncategorized[j].Title + uncategorized[j].URL)
		})
		for _, f := range uncategorized {
			top = append(top, outline{
				Text:   firstNonEmpty(f.Title, f.URL),
				Title:  f.Title,
				Type:   "rss",
				XMLURL: f.URL,
			})
		}
	}

	doc := document{
		Version: "1.0",
		Head: head{
			Title: title,
		},
		Body: body{
			Outlines: top,
		},
	}

	return xml.MarshalIndent(doc, "", "  ")
}

// Parse 解析 OPML，返回订阅项列表
func Parse(data []byte) ([]Item, error) {
	var doc document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var items []Item
	for _, o := range doc.Body.Outlines {
		walkOutline(o, "", &items)
	}
	return items, nil
}

func walkOutline(o outline, parentCategory string, items *[]Item) {
	if o.XMLURL != "" {
		*items = append(*items, Item{
			Category: parentCategory,
			URL:      o.XMLURL,
			Title:    firstNonEmpty(o.Title, o.Text, o.XMLURL),
		})
		return
	}
	if len(o.Outlines) == 0 {
		return
	}
	name := firstNonEmpty(o.Title, o.Text)
	if name == "" {
		name = parentCategory
	}
	for _, child := range o.Outlines {
		walkOutline(child, name, items)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

