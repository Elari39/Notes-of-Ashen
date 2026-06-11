package middleware

import (
	"context"
	"net/http"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type redisLimiter interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

type RateLimitMiddleware struct {
	redisClient redisLimiter
	name        string
	limit       int64
	window      time.Duration
}

func NewRateLimitMiddleware(redisClient *redis.Client, name string, limit int64, window time.Duration) *RateLimitMiddleware {
	var limiter redisLimiter
	if redisClient != nil {
		limiter = redisClient
	}
	return &RateLimitMiddleware{
		redisClient: limiter,
		name:        name,
		limit:       limit,
		window:      window,
	}
}

const rateLimitRedisTimeout = 200 * time.Millisecond

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.redisClient == nil {
			logx.Errorf("rate limit skipped: redis client is nil, name=%s", m.name)
			next(w, r)
			return
		}

		ip := basehandler.Meta(r).IP
		key := security.RateLimitKey(m.name, ip)
		redisCtx, cancel := context.WithTimeout(r.Context(), rateLimitRedisTimeout)
		count, err := m.redisClient.Incr(redisCtx, key).Result()
		cancel()
		if err != nil {
			logx.Errorf("rate limit skipped: redis incr failed, name=%s, err=%v", m.name, err)
			next(w, r)
			return
		}
		if count == 1 {
			redisCtx, cancel = context.WithTimeout(r.Context(), rateLimitRedisTimeout)
			err = m.redisClient.Expire(redisCtx, key, m.window).Err()
			cancel()
			if err != nil {
				logx.Errorf("rate limit expire failed, name=%s, err=%v", m.name, err)
				next(w, r)
				return
			}
		}
		if count > m.limit {
			response.Error(w, apperrors.TooManyRequests("too many requests"))
			return
		}
		next(w, r)
	}
}
