package router

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/handler"
	"zzdzz-blog/server/internal/service"
	jwtutil "zzdzz-blog/server/pkg/jwt"
)

// chainedAdmin = RequireAuth + RequireAdmin 的复合中间件, 用于后台写操作/管理类接口
func chainedAdmin(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先复用 RequireAuth 的解析+注入逻辑
		h := c.GetHeader("Authorization")
		if !authHasBearer(h) {
			handler.RequireAuth(cfg)(c)
			return
		}
		// 已经有合法 token, 直接读 claims 判断 admin
		claims, err := jwtutil.Parse(cfg.Secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			handler.RequireAuth(cfg)(c)
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		if !claims.IsAdmin {
			handler.RequireAdmin()(c)
			return
		}
		c.Next()
	}
}

func authHasBearer(h string) bool {
	return len(h) >= 7 && h[:7] == "Bearer "
}

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
	aiSvc := service.NewAIService(db)
	ai := handler.NewAIHandler(&cfg.AI, aiSvc)
	userSvc := service.NewUserService(db)
	userH := handler.NewUserHandler(userSvc)

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
			// 分类是 admin 后台资源, 非 admin 不允许改
			categories.POST("", chainedAdmin(&cfg.JWT), cat.Create)
			categories.PUT("/:id", chainedAdmin(&cfg.JWT), cat.Update)
			categories.DELETE("/:id", chainedAdmin(&cfg.JWT), cat.Delete)
		}

		articles := api.Group("/articles")
		{
			// List / Get 走 OptionalAuth: 公开可读, 带 token 时 admin 可看全部可见性; 非 admin 与匿名一致只看 public
			articles.GET("", handler.OptionalAuth(&cfg.JWT), art.List)
			articles.GET("/:id", handler.OptionalAuth(&cfg.JWT), art.Get)
			// 写操作: 必须 admin
			articles.POST("", chainedAdmin(&cfg.JWT), art.Create)
			articles.PUT("/:id", chainedAdmin(&cfg.JWT), art.Update)
			articles.DELETE("/:id", chainedAdmin(&cfg.JWT), art.Delete)
		}

		aiGroup := api.Group("/ai")
		aiGroup.Use(handler.RequireAuth(&cfg.JWT))
		{
			// AI 对话是登录用户都可用的功能, 不限 admin
			aiGroup.GET("/conversations", ai.ListConversations)
			aiGroup.POST("/conversations", ai.CreateConversation)
			aiGroup.PATCH("/conversations/:id", ai.RenameConversation)
			aiGroup.DELETE("/conversations/:id", ai.DeleteConversation)
			aiGroup.GET("/conversations/:id/messages", ai.ListMessages)
			aiGroup.POST("/conversations/:id/messages", ai.SendMessage)

			// 兼容旧版（无状态）
			aiGroup.POST("/chat", ai.Chat)
		}

		users := api.Group("/users")
		users.Use(chainedAdmin(&cfg.JWT))
		{
			users.GET("", userH.List)
			users.POST("", userH.Create)
			users.PUT("/:id/password", userH.ChangePassword)
			users.PATCH("/:id/active", userH.SetActive)
		}
	}

	return r
}