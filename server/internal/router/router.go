package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/handler"
	"zzdzz-blog/server/internal/service"
)

func New(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 前端静态文件 (web/dist 在二进制同级目录)
	r.Static("/assets", "./web/assets")
	r.StaticFile("/favicon.svg", "./web/favicon.svg")
	r.StaticFile("/icons.svg", "./web/icons.svg")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	// SPA fallback: 非 /api 的路径都返回 index.html,让前端路由接管
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" {
			c.JSON(404, gin.H{"code": 404, "message": "route not found"})
			return
		}
		c.File("./web/index.html")
	})

	corsCfg := cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsCfg))

	health := handler.NewHealthHandler(db)
	authSvc := service.NewAuthService(db, &cfg.JWT)
	auth := handler.NewAuthHandler(authSvc, &cfg.JWT)
	catSvc := service.NewCategoryService(db)
	cat := handler.NewCategoryHandler(catSvc)
	artSvc := service.NewArticleService(db)
	art := handler.NewArticleHandler(artSvc)
	ai := handler.NewAIHandler(&cfg.AI)

	api := r.Group("/api/v1")
	{
		api.GET("/ping", health.Ping)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", auth.Login)

			protected := authGroup.Group("")
			protected.Use(handler.RequireAuth(&cfg.JWT))
			{
				protected.GET("/me", auth.Me)
			}
		}

		categories := api.Group("/categories")
		{
			categories.GET("", cat.List)
			categories.POST("", handler.RequireAuth(&cfg.JWT), cat.Create)
			categories.PUT("/:id", handler.RequireAuth(&cfg.JWT), cat.Update)
			categories.DELETE("/:id", handler.RequireAuth(&cfg.JWT), cat.Delete)
		}

		articles := api.Group("/articles")
		{
			articles.GET("", art.List)
			articles.GET("/:id", art.Get)
			articles.POST("", handler.RequireAuth(&cfg.JWT), art.Create)
			articles.PUT("/:id", handler.RequireAuth(&cfg.JWT), art.Update)
			articles.DELETE("/:id", handler.RequireAuth(&cfg.JWT), art.Delete)
		}

		aiGroup := api.Group("/ai")
		aiGroup.Use(handler.RequireAuth(&cfg.JWT))
		{
			aiGroup.POST("/chat", ai.Chat)
		}
	}

	return r
}