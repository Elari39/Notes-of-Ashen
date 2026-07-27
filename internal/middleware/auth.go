package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
)

type UserFinder interface {
	FindUserByID(ctx context.Context, id uint64) (*model.User, error)
	UserTokenVersion(ctx context.Context, id uint64) (uint64, error)
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
	redis        *redis.Client
}

func (m *AuthMiddleware) WithTokenCutoff(client *redis.Client) *AuthMiddleware {
	m.redis = client
	return m
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
		ctx, err := m.authenticate(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		next(w, r.WithContext(ctx))
	}
}

// HandleOptional 在 Access Token 有效时注入用户上下文；缺失、过期或无效的
// Access Token 会继续执行下一个 Handler。它只用于注销等可以由其他凭据完成
// 的会话收尾操作，普通受保护接口必须继续使用 Handle。
func (m *AuthMiddleware) HandleOptional(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := m.authenticate(r)
		if err != nil {
			next(w, r)
			return
		}
		next(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) authenticate(r *http.Request) (context.Context, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, apperrors.Unauthorized("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, apperrors.Unauthorized("invalid authorization header")
	}

	claims, err := m.tokenManager.ParseAccessToken(parts[1])
	if err != nil {
		return nil, apperrors.Unauthorized("invalid or expired token")
	}
	if m.tokenIssuedBeforeCutoff(r.Context(), claims) {
		return nil, apperrors.Unauthorized("invalid or expired token")
	}
	currentVersion, err := m.users.UserTokenVersion(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, apperrors.Unauthorized("invalid or expired token")
		}
		return nil, err
	}
	if claims.TokenVersion != currentVersion {
		return nil, apperrors.Unauthorized("invalid or expired token")
	}

	role, status, err := m.resolveUser(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, apperrors.Unauthorized("invalid or expired token")
		}
		return nil, err
	}
	if status != "active" {
		return nil, apperrors.Forbidden("user is disabled")
	}

	return authutil.WithUser(r.Context(), claims.UserID, role), nil
}

func (m *AuthMiddleware) tokenIssuedBeforeCutoff(ctx context.Context, claims *authutil.Claims) bool {
	if claims.IssuedAt == nil {
		return false
	}
	cutoff := security.AccessTokensNotBefore()
	if m.redis == nil {
		return cutoff > 0 && claims.IssuedAt.Unix() <= cutoff
	}
	raw, err := m.redis.Get(ctx, security.AccessTokensNotBeforeKey).Result()
	if err == nil {
		if redisCutoff, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && redisCutoff > cutoff {
			cutoff = redisCutoff
		}
	}
	return cutoff > 0 && claims.IssuedAt.Unix() <= cutoff
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
