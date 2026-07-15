package analytics

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	analyticslogic "notes-of-ashen/internal/logic/analytics"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func OverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := analyticslogic.Overview(r.Context(), svcCtx, rangeReq(r))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ArticlesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := analyticslogic.Articles(r.Context(), svcCtx, rangeReq(r), basehandler.Query(r, "q"), page, size)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func DetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := analyticslogic.Detail(r.Context(), svcCtx, id, rangeReq(r))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func rangeReq(r *http.Request) types.AnalyticsRangeReq {
	return types.AnalyticsRangeReq{From: basehandler.Query(r, "from"), To: basehandler.Query(r, "to")}
}
