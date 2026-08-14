package site

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	sitelogic "notes-of-ashen/internal/logic/site"
	"notes-of-ashen/internal/svc"
)

// HealthzHandler 暴露 /healthz，返回整体存活状态（不输出依赖明细，避免信息泄露）。
// 无需鉴权，供 Docker healthcheck 与部署探针使用。
func HealthzHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := sitelogic.Health(r.Context(), svcCtx)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(sitelogic.HTTPStatus(report))
		if err := json.NewEncoder(w).Encode(report); err != nil {
			logx.Errorf("healthz: encode response failed: %v", err)
		}
	}
}
