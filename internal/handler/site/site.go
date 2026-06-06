package site

import (
	"net/http"

	sitelogic "notes-of-ashen/internal/logic/site"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func SettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := sitelogic.Settings(r.Context(), svcCtx)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateSiteSettingsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := sitelogic.UpdateSettings(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}
