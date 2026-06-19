package traffic

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	trafficlogic "notes-of-ashen/internal/logic/traffic"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func VisitHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TrafficVisitReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := trafficlogic.Visit(r.Context(), svcCtx, req, basehandler.Meta(r, basehandler.ForwardedOptions{
			TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs,
		})); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}
