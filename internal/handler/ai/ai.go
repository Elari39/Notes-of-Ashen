package ai

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	ailogic "notes-of-ashen/internal/logic/ai"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func SettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := ailogic.Settings(r.Context(), svcCtx)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateAISettingsReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := ailogic.UpdateSettings(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}
