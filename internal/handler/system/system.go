package system

import (
	"net/http"
	"strings"

	basehandler "notes-of-ashen/internal/httphelper"
	systemlogic "notes-of-ashen/internal/logic/system"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
)

func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		force := strings.EqualFold(basehandler.Query(r, "refresh"), "true")
		resp, err := systemlogic.Health(r.Context(), svcCtx, force)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
