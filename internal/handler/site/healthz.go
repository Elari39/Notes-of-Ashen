package site

import (
	"encoding/json"
	"net/http"

	sitelogic "notes-of-ashen/internal/logic/site"
	"notes-of-ashen/internal/svc"
)

// HealthzHandler 暴露 /healthz，返回各依赖存活状态。
// 无需鉴权，供 Docker healthcheck 与部署探针使用。
func HealthzHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := sitelogic.Health(r.Context(), svcCtx)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(sitelogic.HTTPStatus(report))
		_ = json.NewEncoder(w).Encode(report)
	}
}
