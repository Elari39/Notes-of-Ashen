package handler

import (
	"net/http"

	adminhandler "notes-of-ashen/internal/handler/admin"
	articlehandler "notes-of-ashen/internal/handler/article"
	authhandler "notes-of-ashen/internal/handler/auth"
	categoryhandler "notes-of-ashen/internal/handler/category"
	sitehandler "notes-of-ashen/internal/handler/site"
	taghandler "notes-of-ashen/internal/handler/tag"
	userhandler "notes-of-ashen/internal/handler/user"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	authMiddleware := middleware.NewAuthMiddleware(svcCtx.Tokens)
	authRequired := func(handler http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Handle(handler)
	}

	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/register", Handler: authhandler.RegisterHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/login", Handler: authhandler.LoginHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Handler: authhandler.RefreshHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/articles", Handler: articlehandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/articles/:id", Handler: articlehandler.DetailHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/categories", Handler: categoryhandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/tags", Handler: taghandler.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/site/settings", Handler: sitehandler.SettingsHandler(svcCtx)},
	})

	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/logout", Handler: authRequired(authhandler.LogoutHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/users/me", Handler: authRequired(userhandler.MeHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/users/me", Handler: authRequired(userhandler.UpdateMeHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/users/me/password", Handler: authRequired(userhandler.ChangePasswordHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/v1/articles", Handler: authRequired(articlehandler.CreateHandler(svcCtx))},
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
		{Method: http.MethodGet, Path: "/api/v1/admin/users", Handler: authRequired(adminhandler.ListUsersHandler(svcCtx))},
		{Method: http.MethodPatch, Path: "/api/v1/admin/users/:id/status", Handler: authRequired(adminhandler.UpdateUserStatusHandler(svcCtx))},
		{Method: http.MethodGet, Path: "/api/v1/admin/logs", Handler: authRequired(adminhandler.ListLogsHandler(svcCtx))},
		{Method: http.MethodPut, Path: "/api/v1/admin/site/settings", Handler: authRequired(sitehandler.UpdateSettingsHandler(svcCtx))},
	})
}
