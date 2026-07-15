package site

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	sitelogic "notes-of-ashen/internal/logic/site"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func SettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := sitelogic.Settings(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ProjectsPageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := sitelogic.ProjectsPage(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func RSSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := sitelogic.RSS(r.Context(), svcCtx, basehandler.RequestBaseURL(r, basehandler.ForwardedOptions{
			TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs,
		}))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(body)
	}
}

func SitemapHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := sitelogic.Sitemap(r.Context(), svcCtx, basehandler.RequestBaseURL(r, basehandler.ForwardedOptions{
			TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs,
		}))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(body)
	}
}

func AdminProjectsPageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := sitelogic.AdminProjectsPage(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateProjectsPageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateProjectsPageReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := sitelogic.UpdateProjectsPage(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateSiteSettingsReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := sitelogic.UpdateSettings(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
