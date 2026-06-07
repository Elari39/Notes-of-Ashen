package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/model"
)

type UserFinder interface {
	FindUserByID(ctx context.Context, id uint64) (*model.User, error)
}

type AuthMiddleware struct {
	tokenManager *authutil.Manager
	users        UserFinder
}

func NewAuthMiddleware(tokenManager *authutil.Manager, users UserFinder) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tokenManager, users: users}
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

		user, err := m.users.FindUserByID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				response.Error(w, apperrors.Unauthorized("invalid or expired token"))
				return
			}
			response.Error(w, err)
			return
		}
		if user.Status != "active" {
			response.Error(w, apperrors.Forbidden("user is disabled"))
			return
		}

		ctx := authutil.WithUser(r.Context(), user.ID, user.Role)
		next(w, r.WithContext(ctx))
	}
}
