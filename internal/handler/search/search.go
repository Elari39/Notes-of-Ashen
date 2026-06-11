package search

import (
	"net/http"

	searchlogic "notes-of-ashen/internal/logic/search"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
)

func ReindexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := searchlogic.Reindex(r.Context(), svcCtx)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}
