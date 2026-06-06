package middleware

import (
	"net/http"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
)

type AuthMiddleware struct {
	tokenManager *authutil.Manager
}

func NewAuthMiddleware(tokenManager *authutil.Manager) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tokenManager}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, apperrors.Unauthorized("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(w, apperrors.Unauthorized("invalid authorization header"))
			return
		}

		claims, err := m.tokenManager.ParseAccessToken(parts[1])
		if err != nil {
			response.Error(w, apperrors.Unauthorized("invalid or expired token"))
			return
		}

		ctx := authutil.WithUser(r.Context(), claims.UserID, claims.Role)
		next(w, r.WithContext(ctx))
	}
}
