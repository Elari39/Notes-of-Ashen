package search

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	searchlogic "notes-of-ashen/internal/logic/search"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
)

func SuggestionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := basehandler.QueryInt(r, "limit", 8)
		resp, err := searchlogic.Suggestions(r.Context(), svcCtx, basehandler.Query(r, "q"), limit)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ReindexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := searchlogic.Reindex(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
