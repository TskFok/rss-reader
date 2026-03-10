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
	db *gorm.DB
	fp *gofeed.Parser
}

// NewRSSService 创建 RSS 服务
func NewRSSService(db *gorm.DB) *RSSService {
	return &RSSService{db: db, fp: gofeed.NewParser()}
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
	for _, item := range parsed.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			continue
		}
		var exists int64
		s.db.Model(&models.Article{}).Where("feed_id = ? AND guid = ?", feed.ID, guid).Count(&exists)
		if exists > 0 {
			continue
		}
		var pubAt *time.Time
		if item.PublishedParsed != nil {
			pubAt = item.PublishedParsed
		}
		content := ""
		if item.Content != "" {
			content = item.Content
		} else if item.Description != "" {
			content = item.Description
		}
		article := models.Article{
			FeedID:      feed.ID,
			GUID:        guid,
			Title:       item.Title,
			Link:        item.Link,
			Content:     content,
			PublishedAt: pubAt,
		}
		if err := s.db.Create(&article).Error; err != nil {
			return err
		}
	}
	return s.db.Model(feed).Update("last_fetched_at", now).Error
}
