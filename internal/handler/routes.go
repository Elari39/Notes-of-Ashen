package handler

import (
	"net/http"
	"time"

	adminhandler "notes-of-ashen/internal/handler/admin"
	aihandler "notes-of-ashen/internal/handler/ai"
	articlehandler "notes-of-ashen/internal/handler/article"
	authhandler "notes-of-ashen/internal/handler/auth"
	categoryhandler "notes-of-ashen/internal/handler/category"
	sitehandler "notes-of-ashen/internal/handler/site"
	taghandler "notes-of-ashen/internal/handler/tag"
	traffichandler "notes-of-ashen/internal/handler/traffic"
	userhandler "notes-of-ashen/internal/handler/user"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	authMiddleware := middleware.NewAuthMiddleware(svcCtx.Tokens, svcCtx.Store)
	loginRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "auth_login", 5, time.Minute)
	verifyCodeRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "verify_code_send", 5, time.Minute)
	resetPasswordRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "password_reset", 5, time.Minute)
	trafficRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "traffic_visit", 120, time.Minute)
	articleLikeRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "article_like", 60, time.Minute)
	aiRateLimit := middleware.NewRateLimitMiddleware(svcCtx.Redis, "ai_assist", 20, time.Minute)
	authRequired := func(handler http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Handle(handler)
	}

	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/rss.xml", Handler: sitehandler.RSSHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/sitemap.xml", Handler: sitehandler.SitemapHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/captcha", Handler: authhandler.CaptchaHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/verify-code/send", Handler: verifyCodeRateLimit.Handle(authhandler.SendVerifyCodeHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/register", Handler: authhandler.RegisterHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/login", Handler: loginRateLimit.Handle(authhandler.LoginHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/password/reset", Handler: resetPasswordRateLimit.Handle(authhandler.ResetPasswordHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Handler: authhandler.RefreshHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/traffic/visit", Handler: trafficRateLimit.Handle(traffichandler.VisitHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/articles", Handler: articlehandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id/context", Handler: articlehandler.ContextHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id", Handler: articlehandler.DetailHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/articles/:id/like", Handler: articleLikeRateLimit.Handle(articlehandler.LikeHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/categories", Handler: categoryhandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/tags", Handler: taghandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/settings", Handler: sitehandler.SettingsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/resume", Handler: sitehandler.ResumePageHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/projects", Handler: sitehandler.ProjectsPageHandler(svcCtx)},
	})

	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/logout", Handler: authRequired(authhandler.LogoutHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/users/me", Handler: authRequired(userhandler.MeHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/users/me", Handler: authRequired(userhandler.UpdateMeHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/users/me/verify-code/send", Handler: verifyCodeRateLimit.Handle(authRequired(userhandler.SendVerifyCodeHandler(svcCtx)))},
		{Method: http.MethodPut, Path: "/api/v1/users/me/password", Handler: authRequired(userhandler.ChangePasswordHandler(svcCtx))},
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
		{Method: http.MethodGet, Path: "/api/v1/admin/stats", Handler: authRequired(adminhandler.StatsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/ai/settings", Handler: authRequired(aihandler.SettingsHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/ai/settings", Handler: authRequired(aihandler.UpdateSettingsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/users", Handler: authRequired(adminhandler.ListUsersHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/users/:id/status", Handler: authRequired(adminhandler.UpdateUserStatusHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/users/:id/role", Handler: authRequired(adminhandler.UpdateUserRoleHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/logs", Handler: authRequired(adminhandler.ListLogsHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/settings", Handler: authRequired(sitehandler.UpdateSettingsHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/site/resume", Handler: authRequired(sitehandler.AdminResumePageHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/resume", Handler: authRequired(sitehandler.UpdateResumePageHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/site/projects", Handler: authRequired(sitehandler.AdminProjectsPageHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/projects", Handler: authRequired(sitehandler.UpdateProjectsPageHandler(svcCtx))},
	})
}
