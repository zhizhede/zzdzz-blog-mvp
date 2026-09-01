package router

import (
	"os"
	"path/filepath"
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

func New(db *gorm.DB, cfg *config.Config) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 前端静态文件 (web/dist 在二进制同级目录)
	r.Static("/assets", "./web/assets")
	r.StaticFile("/icons.svg", "./web/icons.svg")

	// 站点设置: 自定义 favicon (服务动态文件 + 首页版本注入)
	iconSvc, err := service.NewIconService(cfg.Site.IconDir)
	if err != nil {
		return nil, err
	}
	site := handler.NewSiteHandler(iconSvc)

	// favicon 系列一律走 site handler: 有自定义图标优先, 否则回退 web/ 内置
	faviconFiles := []string{
		"favicon.svg", "favicon.ico", "apple-touch-icon.png",
		"favicon-32.png", "favicon-48.png", "favicon-64.png",
		"favicon-128.png", "favicon-256.png", "favicon-512.png",
	}
	for _, f := range faviconFiles {
		r.GET("/"+f, site.ServeIconFile("./web", f))
	}
	r.GET("/", site.ServeIndex("./web"))
	// SPA fallback: 非 /api 的路径优先匹配 web/ 下的真实文件 (favicon 等),否则返回 index.html
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"code": 404, "message": "route not found"})
			return
		}
		p := filepath.Join("./web", filepath.Clean("/"+c.Request.URL.Path))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			c.File(p)
			return
		}
		site.ServeIndex("./web")(c)
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
	tagSvc := service.NewTagService(db)
	tag := handler.NewTagHandler(tagSvc)
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
				// 自助改密: 登录用户改自己的密码(强制 actorID == targetID)
				protected.PUT("/password", auth.ChangeOwnPassword)
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
			// List / Get 走 OptionalAuth: 公开可读; admin 可看全部可见性;
			// 自己 (handler 按 author_id 分支) 也可看自己的 private/draft; 其他人按 public
			articles.GET("", handler.OptionalAuth(&cfg.JWT), art.List)
			articles.GET("/:id", handler.OptionalAuth(&cfg.JWT), art.Get)
			// 带标签的详情/编辑视图
			articles.GET("/:id/full", handler.OptionalAuth(&cfg.JWT), art.GetWithTags)
			// 写操作: 任何登录用户都可调, handler 内部按"admin 或作者本人"判断权限
			articles.POST("", handler.RequireAuth(&cfg.JWT), art.Create)
			articles.PUT("/:id", handler.RequireAuth(&cfg.JWT), art.Update)
			articles.DELETE("/:id", handler.RequireAuth(&cfg.JWT), art.Delete)
			// 草稿自动保存: 仅作者/admin, 仅 draft 文章
			articles.PUT("/:id/autosave", handler.RequireAuth(&cfg.JWT), art.Autosave)
			// 可见性单独改: 任何状态(public/private/draft)都允许, 仅作者/admin
			articles.PATCH("/:id/visibility", handler.RequireAuth(&cfg.JWT), art.SetVisibility)
			// 当前用户自己的 draft 列表
			articles.GET("/autosave/drafts", handler.RequireAuth(&cfg.JWT), art.ListMyDrafts)
		}

		tags := api.Group("/tags")
		{
			// 标签列表公开, 写入需 admin
			tags.GET("", tag.List)
			tags.POST("", chainedAdmin(&cfg.JWT), tag.Create)
			tags.PUT("/:id", chainedAdmin(&cfg.JWT), tag.Update)
			tags.DELETE("/:id", chainedAdmin(&cfg.JWT), tag.Delete)
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

		// 站点设置: 仅 admin
		siteGroup := api.Group("/site")
		siteGroup.Use(chainedAdmin(&cfg.JWT))
		{
			siteGroup.PUT("/icon", site.Upload)
			siteGroup.DELETE("/icon", site.Reset)
			siteGroup.GET("/icon/meta", site.Meta)
		}
	}

	return r, nil
}