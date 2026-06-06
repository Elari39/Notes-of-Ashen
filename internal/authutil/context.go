package authutil

import (
	"context"
	"strconv"

	apperrors "notes-of-ashen/internal/errors"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
	roleKey   contextKey = "role"
)

func WithUser(ctx context.Context, userID uint64, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, roleKey, role)
}

func UserID(ctx context.Context) (uint64, error) {
	switch v := ctx.Value(userIDKey).(type) {
	case uint64:
		return v, nil
	case int64:
		if v <= 0 {
			return 0, apperrors.Unauthorized("invalid user")
		}
		return uint64(v), nil
	case string:
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || id == 0 {
			return 0, apperrors.Unauthorized("invalid user")
		}
		return id, nil
	default:
		return 0, apperrors.Unauthorized("missing user")
	}
}

func Role(ctx context.Context) string {
	role, _ := ctx.Value(roleKey).(string)
	return role
}

func RequireAdmin(ctx context.Context) error {
	if Role(ctx) != "admin" {
		return apperrors.Forbidden("admin permission required")
	}
	return nil
}
