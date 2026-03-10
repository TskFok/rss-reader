package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/models"
	"github.com/ushopal/rss-reader/internal/opml"
	"github.com/ushopal/rss-reader/internal/services"
)

type OPMLHandler struct {
	feedSvc *services.FeedService
	catSvc  *services.CategoryService
}

func NewOPMLHandler(feedSvc *services.FeedService, catSvc *services.CategoryService) *OPMLHandler {
	return &OPMLHandler{feedSvc: feedSvc, catSvc: catSvc}
}

// Export 导出当前用户的订阅为 OPML
// GET /api/feeds/opml
func (h *OPMLHandler) Export(c *gin.Context) {
	userID := middleware.GetUserID(c)
	feeds, err := h.feedSvc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取订阅列表失败"})
		return
	}
	data, err := opml.Generate("RSS 订阅导出", feeds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 OPML 失败"})
		return
	}
	c.Header("Content-Type", "text/xml; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"subscriptions.opml\"")
	c.String(http.StatusOK, xmlHeader()+string(data))
}

// Import 从 OPML 导入订阅
// POST /api/feeds/opml
func (h *OPMLHandler) Import(c *gin.Context) {
	userID := middleware.GetUserID(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未上传文件"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 2*1024*1024))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
		return
	}
	items, err := opml.Parse(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析 OPML 失败"})
		return
	}
	existingCats, _ := h.catSvc.List(userID)
	catMap := make(map[string]*models.FeedCategory)
	for i := range existingCats {
		cat := existingCats[i]
		catMap[cat.Name] = &cat
	}

	var imported, skipped, failed int

	getCategoryID := func(name string) (uint, error) {
		if name == "" {
			return 0, nil
		}
		if c, ok := catMap[name]; ok {
			return c.ID, nil
		}
		cat, err := h.catSvc.Create(userID, services.CreateCategoryRequest{Name: name})
		if err != nil {
			return 0, err
		}
		catMap[name] = cat
		return cat.ID, nil
	}

	for _, it := range items {
		catID, err := getCategoryID(it.Category)
		if err != nil {
			failed++
			continue
		}
		if catID == 0 {
			// 未指定分类，允许默认分类为空（调用创建时会校验）
		}
		expireDays := 90
		req := services.CreateFeedRequest{
			URL:                   it.URL,
			CategoryID:            catID,
			UpdateIntervalMinutes: 60,
			ExpireDays:            &expireDays,
		}
		_, err = h.feedSvc.Create(userID, req)
		if err != nil {
			if err == services.ErrInvalidFeedURL {
				failed++
				continue
			}
			if err.Error() == "订阅已存在" {
				skipped++
				continue
			}
			if err.Error() == "分类不存在" {
				failed++
				continue
			}
			failed++
			continue
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"skipped":  skipped,
		"failed":   failed,
	})
}

func xmlHeader() string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
}

