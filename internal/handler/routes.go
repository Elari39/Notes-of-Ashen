package handler

import (
	"net/http"
	"time"

	adminhandler "notes-of-ashen/internal/handler/admin"
	aihandler "notes-of-ashen/internal/handler/ai"
	analyticshandler "notes-of-ashen/internal/handler/analytics"
	articlehandler "notes-of-ashen/internal/handler/article"
	authhandler "notes-of-ashen/internal/handler/auth"
	backuphandler "notes-of-ashen/internal/handler/backup"
	categoryhandler "notes-of-ashen/internal/handler/category"
	mediahandler "notes-of-ashen/internal/handler/media"
	searchhandler "notes-of-ashen/internal/handler/search"
	sitehandler "notes-of-ashen/internal/handler/site"
	systemhandler "notes-of-ashen/internal/handler/system"
	taghandler "notes-of-ashen/internal/handler/tag"
	traffichandler "notes-of-ashen/internal/handler/traffic"
	userhandler "notes-of-ashen/internal/handler/user"
	basehandler "notes-of-ashen/internal/httphelper"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	forwardedOptions := middlewareForwardedOptions(svcCtx)
	// RequestID 为所有请求注入 X-Request-Id 与请求上下文，供错误日志关联排查。
	server.Use(middleware.RequestID)
	// 安全访问日志只记录请求元数据，禁止输出查询串、Header、Cookie 和请求正文。
	server.Use(middleware.NewAccessLogMiddleware(forwardedOptions).Handle)
	server.Use(middleware.NewMaintenanceMiddleware(svcCtx.Redis).Handle)
	authMiddleware := middleware.NewAuthMiddleware(svcCtx.Tokens, svcCtx.Store).
		WithUserCache(svcCtx.AuthUserCache).
		WithTokenCutoff(svcCtx.Redis)
	loginRateLimit := middleware.NewFailClosedRateLimitMiddleware(svcCtx.Redis, "auth_login", 5, time.Minute, forwardedOptions)
	verifyCodeRateLimit := middleware.NewFailClosedRateLimitMiddleware(svcCtx.Redis, "verify_code_send", 5, time.Minute, forwardedOptions)
	resetPasswordRateLimit := middleware.NewFailClosedRateLimitMiddleware(svcCtx.Redis, "password_reset", 5, time.Minute, forwardedOptions)
	changePasswordRateLimit := middleware.NewFailClosedRateLimitMiddleware(svcCtx.Redis, "password_change", 5, time.Minute, forwardedOptions)
	captchaRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "auth_captcha", 30, time.Minute, forwardedOptions)
	registerRateLimit := middleware.NewFailClosedRateLimitMiddleware(svcCtx.Redis, "auth_register", 5, time.Minute, forwardedOptions)
	trafficRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "traffic_visit", 120, time.Minute, forwardedOptions)
	articleLikeRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "article_like", 60, time.Minute, forwardedOptions)
	aiRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "ai_assist", 20, time.Minute, forwardedOptions)
	authRequired := func(handler http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Handle(handler)
	}

	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/healthz", Handler: sitehandler.HealthzHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/rss.xml", Handler: sitehandler.RSSHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/sitemap.xml", Handler: sitehandler.SitemapHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/media/:key", Handler: mediahandler.FileHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/captcha", Handler: captchaRateLimit.Handle(authhandler.CaptchaHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/verify-code/send", Handler: verifyCodeRateLimit.Handle(authhandler.SendVerifyCodeHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/register", Handler: registerRateLimit.Handle(authhandler.RegisterHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/login", Handler: loginRateLimit.Handle(authhandler.LoginHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/password/reset", Handler: resetPasswordRateLimit.Handle(authhandler.ResetPasswordHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Handler: authhandler.RefreshHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/traffic/visit", Handler: trafficRateLimit.Handle(traffichandler.VisitHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles", Handler: articlehandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/context", Handler: articlehandler.ContextHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/articles/:id/like", Handler: articleLikeRateLimit.Handle(articlehandler.LikeHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id", Handler: articlehandler.DetailHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/categories", Handler: categoryhandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/tags", Handler: taghandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/search/suggestions", Handler: searchhandler.SuggestionsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/settings", Handler: sitehandler.SettingsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/projects", Handler: sitehandler.ProjectsPageHandler(svcCtx)},
	})

	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/logout", Handler: authRequired(authhandler.LogoutHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/users/me", Handler: authRequired(userhandler.MeHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/users/me", Handler: authRequired(userhandler.UpdateMeHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/users/me/verify-code/send", Handler: verifyCodeRateLimit.Handle(authRequired(userhandler.SendVerifyCodeHandler(svcCtx)))},
		{Method: http.MethodPut, Path: "/api/v1/users/me/password", Handler: changePasswordRateLimit.Handle(authRequired(userhandler.ChangePasswordHandler(svcCtx)))},
		{Method: http.MethodPost, Path: "/api/v1/articles", Handler: authRequired(articlehandler.CreateHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/articles/ai/assist", Handler: aiRateLimit.Handle(authRequired(articlehandler.AIAssistHandler(svcCtx)))},
		{Method: http.MethodPost, Path: "/api/v1/articles/import", Handler: authRequired(articlehandler.ImportMarkdownHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/preview", Handler: authRequired(articlehandler.PreviewHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/export", Handler: authRequired(articlehandler.ExportMarkdownHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/versions", Handler: authRequired(articlehandler.ListVersionsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/versions/:versionNo", Handler: authRequired(articlehandler.VersionDetailHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/articles/:id/versions/:versionNo/restore", Handler: authRequired(articlehandler.RestoreVersionHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/articles/:id", Handler: authRequired(articlehandler.UpdateHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/v1/articles/:id", Handler: authRequired(articlehandler.DeleteHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/articles/:id/status", Handler: authRequired(articlehandler.UpdateStatusHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/categories", Handler: authRequired(categoryhandler.CreateHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/categories/:id", Handler: authRequired(categoryhandler.UpdateHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/v1/categories/:id", Handler: authRequired(categoryhandler.DeleteHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/tags", Handler: authRequired(taghandler.CreateHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/tags/:id", Handler: authRequired(taghandler.UpdateHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/v1/tags/:id", Handler: authRequired(taghandler.DeleteHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/articles", Handler: authRequired(articlehandler.AdminListHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/categories", Handler: authRequired(categoryhandler.AdminListHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/tags", Handler: authRequired(taghandler.AdminListHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/media", Handler: authRequired(mediahandler.ListHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/admin/media", Handler: authRequired(mediahandler.UploadHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/media/:id", Handler: authRequired(mediahandler.UpdateHandler(svcCtx))},
		{Method: http.MethodDelete, Path: "/api/v1/admin/media/:id", Handler: authRequired(mediahandler.DeleteHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/analytics/overview", Handler: authRequired(analyticshandler.OverviewHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/analytics/articles", Handler: authRequired(analyticshandler.ArticlesHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/analytics/articles/:id", Handler: authRequired(analyticshandler.DetailHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/system/health", Handler: authRequired(systemhandler.HealthHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/admin/backups/export", Handler: authRequired(backuphandler.ExportHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/admin/backups/restore", Handler: authRequired(backuphandler.RestoreHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/stats", Handler: authRequired(adminhandler.StatsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/ai/settings", Handler: authRequired(aihandler.SettingsHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/ai/settings", Handler: authRequired(aihandler.UpdateSettingsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/admin/ai/models", Handler: aiRateLimit.Handle(authRequired(aihandler.ModelsHandler(svcCtx)))},
		{Method: http.MethodPost, Path: "/api/v1/admin/ai/test", Handler: aiRateLimit.Handle(authRequired(aihandler.TestModelHandler(svcCtx)))},
		{Method: http.MethodGet, Path: "/api/v1/admin/users", Handler: authRequired(adminhandler.ListUsersHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/users/:id/status", Handler: authRequired(adminhandler.UpdateUserStatusHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/users/:id/role", Handler: authRequired(adminhandler.UpdateUserRoleHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/logs", Handler: authRequired(adminhandler.ListLogsHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/admin/search/reindex", Handler: authRequired(searchhandler.ReindexHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/settings", Handler: authRequired(sitehandler.UpdateSettingsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/site/projects", Handler: authRequired(sitehandler.AdminProjectsPageHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/projects", Handler: authRequired(sitehandler.UpdateProjectsPageHandler(svcCtx))},
	})
}

func middlewareForwardedOptions(svcCtx *svc.ServiceContext) basehandler.ForwardedOptions {
	return basehandler.ForwardedOptions{TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs}
}
