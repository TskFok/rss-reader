package services

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/ushopal/rss-reader/internal/models"
	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

var (
	ErrInvalidFeedURL = errors.New("无效的 RSS 地址")
)

// RSSService RSS 抓取服务
type RSSService struct {
	db        *gorm.DB
	fp        *gofeed.Parser
	articleAI *ArticleAIProcessor
}

// NewRSSService 创建 RSS 服务
func NewRSSService(db *gorm.DB) *RSSService {
	return &RSSService{db: db, fp: gofeed.NewParser()}
}

// SetArticleAI 设置文章入库后的异步 AI 处理器（可选）
func (s *RSSService) SetArticleAI(p *ArticleAIProcessor) {
	s.articleAI = p
}

// parserWithProxy 返回配置了代理的 Parser，proxyURL 为空则直连
func parserWithProxy(proxyURL string) *gofeed.Parser {
	fp := gofeed.NewParser()
	if proxyURL == "" {
		return fp
	}
	transport := httpTransportWithProxy(proxyURL)
	if transport != nil {
		fp.Client = &http.Client{Transport: transport, Timeout: 30 * time.Second}
	}
	return fp
}

func httpTransportWithProxy(proxyURL string) *http.Transport {
	pu := strings.TrimSpace(proxyURL)
	if pu == "" {
		return nil
	}
	u, err := url.Parse(pu)
	if err != nil {
		return nil
	}
	switch u.Scheme {
	case "http", "https":
		return &http.Transport{
			Proxy:                 http.ProxyURL(u),
			ResponseHeaderTimeout: 15 * time.Second,
		}
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(u, proxy.Direct)
		if err != nil {
			return nil
		}
		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			ResponseHeaderTimeout: 15 * time.Second,
		}
	default:
		return nil
	}
}

// FetchAndParse 抓取并解析 feed，返回标题等信息；proxyURL 为空则直连
func (s *RSSService) FetchAndParse(feedURL string, proxyURL string) (title string, err error) {
	fp := parserWithProxy(proxyURL)
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return "", ErrInvalidFeedURL
	}
	if feed.Title != "" {
		return feed.Title, nil
	}
	return feedURL, nil
}

type feedItemCandidate struct {
	item *gofeed.Item
	raw  string
	guid string
}

// existingArticleGUIDs 批量查询指定 feed 下已存在的文章 guid。
func (s *RSSService) existingArticleGUIDs(feedID uint, guids []string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	if len(guids) == 0 {
		return existing, nil
	}
	var rows []string
	if err := s.db.Model(&models.Article{}).
		Where("feed_id = ? AND guid IN ?", feedID, guids).
		Pluck("guid", &rows).Error; err != nil {
		return nil, err
	}
	for _, g := range rows {
		existing[g] = struct{}{}
	}
	return existing, nil
}

// FetchFeed 抓取 feed 并更新文章；若 feed.Proxy 不为空则通过代理抓取
func (s *RSSService) FetchFeed(feed *models.Feed) error {
	proxyURL := ""
	if feed.Proxy != nil {
		proxyURL = feed.Proxy.URL
	}
	fp := parserWithProxy(proxyURL)
	parsed, err := fp.ParseURL(feed.URL)
	if err != nil {
		return err
	}
	now := time.Now()

	candidates := make([]feedItemCandidate, 0, len(parsed.Items))
	uniqueGUIDs := make([]string, 0, len(parsed.Items))
	seenGUID := make(map[string]struct{})
	for _, item := range parsed.Items {
		raw := strings.TrimSpace(item.GUID)
		if raw == "" {
			raw = strings.TrimSpace(item.Link)
		}
		if raw == "" {
			continue
		}
		guid := models.ArticleGUIDHash(raw)
		if guid == "" {
			continue
		}
		candidates = append(candidates, feedItemCandidate{item: item, raw: raw, guid: guid})
		if _, ok := seenGUID[guid]; !ok {
			seenGUID[guid] = struct{}{}
			uniqueGUIDs = append(uniqueGUIDs, guid)
		}
	}

	existing, err := s.existingArticleGUIDs(feed.ID, uniqueGUIDs)
	if err != nil {
		return err
	}

	for _, c := range candidates {
		if _, ok := existing[c.guid]; ok {
			continue
		}
		var pubAt *time.Time
		if c.item.PublishedParsed != nil {
			pubAt = c.item.PublishedParsed
		}
		content := ""
		if c.item.Content != "" {
			content = c.item.Content
		} else if c.item.Description != "" {
			content = c.item.Description
		}
		article := models.Article{
			FeedID:      feed.ID,
			GUID:        c.guid,
			GUIDRaw:     c.raw,
			Title:       c.item.Title,
			Link:        c.item.Link,
			Content:     content,
			PublishedAt: pubAt,
		}
		if err := s.db.Create(&article).Error; err != nil {
			return err
		}
		existing[c.guid] = struct{}{}
		if s.articleAI != nil {
			s.articleAI.Enqueue(feed, article.ID)
		}
	}
	return s.db.Model(feed).Update("last_fetched_at", now).Error
}
