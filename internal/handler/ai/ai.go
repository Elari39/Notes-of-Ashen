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
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateAISettingsReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.StandardJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := ailogic.UpdateSettings(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ModelsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AIConnectionReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.StandardJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := ailogic.Models(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func TestModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AIModelTestReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.StandardJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := ailogic.TestModel(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
