package main

import (
	"fmt"
	iofs "io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/config"
	"github.com/ushopal/rss-reader/internal/database"
	"github.com/ushopal/rss-reader/internal/handlers"
	"github.com/ushopal/rss-reader/internal/logger"
	"github.com/ushopal/rss-reader/internal/middleware"
	"github.com/ushopal/rss-reader/internal/scheduler"
	"github.com/ushopal/rss-reader/internal/services"
)

func main() {
	cfgPath := "config.yaml"
	if p := os.Getenv("CONFIG"); p != "" {
		cfgPath = p
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) && cfgPath == "config.yaml" {
		cfgPath = "config.example.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}
	logger.Init(cfg.Log.Level)
	db, err := database.Init(cfg.Database.DSN)
	if err != nil {
		logger.Fatalf("init db: %v", err)
	}
	authSvc := services.NewAuthService(db, cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.SuperAdmin.Username)
	rssSvc := services.NewRSSService(db)
	feedSvc := services.NewFeedService(db, rssSvc)
	categorySvc := services.NewCategoryService(db)
	proxySvc := services.NewProxyService(db)
	articleSvc := services.NewArticleService(db)
	adminSvc := services.NewAdminService(db)
	opmlHandler := handlers.NewOPMLHandler(feedSvc, categorySvc)
	feishuAPI := services.NewFeishuHTTPClient(cfg.Feishu.AppID, cfg.Feishu.AppSecret, cfg.Feishu.Redirect)
	feishuAuthSvc := services.NewFeishuAuthService(db)
	feishuHandler := handlers.NewFeishuHandler(&cfg.Feishu, feishuAPI, authSvc, feishuAuthSvc)

	sched := scheduler.New(db, rssSvc, articleSvc, 3)
	sched.Start()
	defer sched.Stop()

	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	var r *gin.Engine
	if cfg.Server.Debug {
		r = gin.Default() // 含 Logger + Recovery，会输出请求日志
	} else {
		r = gin.New()
		r.Use(gin.Recovery()) // 仅 Recovery，不输出请求日志
	}
	// 避免 Gin 的自动路径重定向（在部分环境下会对 "/" 返回 Location: "./" 导致循环重定向）
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.RemoveExtraSlash = false

	api := r.Group("/api")
	{
		api.POST("/auth/register", handlers.NewAuthHandler(authSvc).Register)
		api.POST("/auth/login", handlers.NewAuthHandler(authSvc).Login)
		api.GET("/auth/feishu/login-url", feishuHandler.LoginURL)
		api.GET("/auth/feishu/login", feishuHandler.LoginRedirect)
		api.GET("/auth/feishu/callback", feishuHandler.Callback)

		auth := api.Group("")
		auth.Use(middleware.Auth(authSvc))
		{
				auth.GET("/feeds/opml", opmlHandler.Export)
				auth.POST("/feeds/opml", opmlHandler.Import)

			auth.GET("/categories", handlers.NewCategoryHandler(categorySvc).List)
			auth.POST("/categories", handlers.NewCategoryHandler(categorySvc).Create)
			auth.PUT("/categories/:id", handlers.NewCategoryHandler(categorySvc).Update)
			auth.DELETE("/categories/:id", handlers.NewCategoryHandler(categorySvc).Delete)

			auth.GET("/proxies", handlers.NewProxyHandler(proxySvc).List)
			auth.POST("/proxies", handlers.NewProxyHandler(proxySvc).Create)
			auth.PUT("/proxies/:id", handlers.NewProxyHandler(proxySvc).Update)
			auth.DELETE("/proxies/:id", handlers.NewProxyHandler(proxySvc).Delete)

			auth.GET("/feeds", handlers.NewFeedHandler(feedSvc).List)
			auth.POST("/feeds", handlers.NewFeedHandler(feedSvc).Create)
			auth.PUT("/feeds/:id", handlers.NewFeedHandler(feedSvc).Update)
			auth.DELETE("/feeds/:id", handlers.NewFeedHandler(feedSvc).Delete)
			auth.GET("/articles", handlers.NewArticleHandler(articleSvc).List)
			auth.PUT("/articles/:id/read", handlers.NewArticleHandler(articleSvc).MarkRead)
			auth.PUT("/articles/:id/favorite", handlers.NewArticleHandler(articleSvc).ToggleFavorite)

			admin := auth.Group("/admin")
			admin.Use(middleware.RequireSuperAdmin(db))
			{
				admin.GET("/users", handlers.NewAdminHandler(adminSvc).ListUsers)
				admin.PUT("/users/:id/unlock", handlers.NewAdminHandler(adminSvc).UnlockUser)
				admin.GET("/users/:id/feishu/bind-url", feishuHandler.BindURL)
			}
		}
	}

	registerStatic(r)

	addr := ":" + fmt.Sprint(cfg.Server.Port)
	logger.Info("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		logger.Fatal(err)
	}
}

func registerStatic(r *gin.Engine) {
	fs := getStaticFS()
	indexHTML, indexErr := iofs.ReadFile(fs, "index.html")
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			// 直接返回 index.html，避免 FileServer 对目录/相对路径的 301 重定向
			if indexErr == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
				return
			}
			// 兜底：如果读取失败，再尝试走静态文件逻辑
			path = "index.html"
		}
		f, err := fs.Open(path)
		if err != nil {
			// SPA 路由回退：返回 index.html
			if indexErr == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "static index missing"})
			return
		}
		f.Close()
		c.FileFromFS(path, http.FS(fs))
	})
}

