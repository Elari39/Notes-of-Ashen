package middleware

import (
	"net/http"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
)

type MaintenanceMiddleware struct{ redis *redis.Client }

func NewMaintenanceMiddleware(client *redis.Client) *MaintenanceMiddleware {
	return &MaintenanceMiddleware{redis: client}
}

func (m *MaintenanceMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.URL.Path == "/api/v1/admin/backups/restore" {
			next(w, r)
			return
		}
		if security.RestoreInProgress() {
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("site restore is in progress"))
			return
		}
		if m.redis != nil {
			active, err := m.redis.Exists(r.Context(), security.RestoreMaintenanceKey).Result()
			if err == nil && active > 0 {
				response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("site restore is in progress"))
				return
			}
		}
		next(w, r)
	}
}
