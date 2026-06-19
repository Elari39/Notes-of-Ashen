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
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

type RateLimitMiddleware struct {
	redisClient redisLimiter
	name        string
	limit       int64
	window      time.Duration
	forwarded   basehandler.ForwardedOptions
}

func NewRateLimitMiddleware(redisClient *redis.Client, name string, limit int64, window time.Duration, forwarded ...basehandler.ForwardedOptions) *RateLimitMiddleware {
	var limiter redisLimiter
	if redisClient != nil {
		limiter = redisClient
	}
	options := basehandler.ForwardedOptions{}
	if len(forwarded) > 0 {
		options = forwarded[0]
	}
	return &RateLimitMiddleware{
		redisClient: limiter,
		name:        name,
		limit:       limit,
		window:      window,
		forwarded:   options,
	}
}

const rateLimitRedisTimeout = 200 * time.Millisecond

const rateLimitScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.redisClient == nil {
			logx.Errorf("rate limit skipped: redis client is nil, name=%s", m.name)
			next(w, r)
			return
		}

		ip := basehandler.Meta(r, m.forwarded).IP
		key := security.RateLimitKey(m.name, ip)
		redisCtx, cancel := context.WithTimeout(r.Context(), rateLimitRedisTimeout)
		count, err := m.redisClient.Eval(redisCtx, rateLimitScript, []string{key}, m.window.Milliseconds()).Int64()
		cancel()
		if err != nil {
			logx.Errorf("rate limit skipped: redis eval failed, name=%s, err=%v", m.name, err)
			next(w, r)
			return
		}
		if count > m.limit {
			response.Error(w, apperrors.TooManyRequests("too many requests"))
			return
		}
		next(w, r)
	}
}
