package middleware

import (
	"context"
	"net/http"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
)

// maintenanceRedisTimeout 限制跨进程恢复标记的可选查询时长。
// Redis 故障不能把敏感路由（尤其是 fail-closed 限流）拖到完整 HTTP 请求超时。
const maintenanceRedisTimeout = 200 * time.Millisecond

type MaintenanceMiddleware struct{ redis *redis.Client }

func NewMaintenanceMiddleware(client *redis.Client) *MaintenanceMiddleware {
	return &MaintenanceMiddleware{redis: client}
}

func (m *MaintenanceMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		if security.RestoreInProgress() {
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("site restore is in progress"))
			return
		}
		if m.redis != nil {
			checkCtx, cancel := context.WithTimeout(r.Context(), maintenanceRedisTimeout)
			active, err := m.redis.Exists(checkCtx, security.RestoreMaintenanceKey).Result()
			cancel()
			if err == nil && active > 0 {
				response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("site restore is in progress"))
				return
			}
		}
		next(w, r)
	}
}
