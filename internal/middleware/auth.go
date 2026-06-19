package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/model"
)

type UserFinder interface {
	FindUserByID(ctx context.Context, id uint64) (*model.User, error)
}

// AuthUserCache 是认证中间件使用的短 TTL 用户快照缓存抽象，
// 避免每个 Bearer 请求都查库。Redis 故障时应返回 miss 而非错误，
// 由中间件降级为直查 DB（fail-open 缓存，不引入新的 500）。
type AuthUserCache interface {
	Get(ctx context.Context, userID uint64) (*authUserSnapshot, bool)
	Set(ctx context.Context, userID uint64, snapshot authUserSnapshot)
	Delete(ctx context.Context, userID uint64)
}

// authUserSnapshot 只缓存鉴权所需的最小字段，避免把 PasswordHash 等写入缓存。
type authUserSnapshot struct {
	Role   string
	Status string
}

const authUserCacheTTL = 30 * time.Second

type AuthMiddleware struct {
	tokenManager *authutil.Manager
	users        UserFinder
	cache        AuthUserCache
}

func NewAuthMiddleware(tokenManager *authutil.Manager, users UserFinder) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tokenManager, users: users}
}

// WithUserCache 注入用户状态短 TTL 缓存。传入 nil 等价于直查 DB。
func (m *AuthMiddleware) WithUserCache(cache AuthUserCache) *AuthMiddleware {
	m.cache = cache
	return m
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.ErrorCtx(r.Context(), w, apperrors.Unauthorized("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.ErrorCtx(r.Context(), w, apperrors.Unauthorized("invalid authorization header"))
			return
		}

		claims, err := m.tokenManager.ParseAccessToken(parts[1])
		if err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.Unauthorized("invalid or expired token"))
			return
		}

		role, status, err := m.resolveUser(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				response.ErrorCtx(r.Context(), w, apperrors.Unauthorized("invalid or expired token"))
				return
			}
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if status != "active" {
			response.ErrorCtx(r.Context(), w, apperrors.Forbidden("user is disabled"))
			return
		}

		ctx := authutil.WithUser(r.Context(), claims.UserID, role)
		next(w, r.WithContext(ctx))
	}
}

// resolveUser 优先读缓存，未命中或缓存不可用时直查 DB 并回填。
// 缓存层自身故障不得导致请求失败，降级为直查 DB。
func (m *AuthMiddleware) resolveUser(ctx context.Context, userID uint64) (string, string, error) {
	if m.cache != nil {
		if snapshot, ok := m.cache.Get(ctx, userID); ok {
			return snapshot.Role, snapshot.Status, nil
		}
	}

	user, err := m.users.FindUserByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	if m.cache != nil {
		m.cache.Set(ctx, userID, authUserSnapshot{Role: user.Role, Status: user.Status})
	}
	return user.Role, user.Status, nil
}

// EvictAuthUserCache 删除指定用户的认证缓存。
// 在禁用/启用用户、修改角色、修改密码等改变鉴权语义的操作后调用。
func EvictAuthUserCache(ctx context.Context, cache AuthUserCache, userID uint64) {
	if cache == nil {
		return
	}
	cache.Delete(ctx, userID)
}
