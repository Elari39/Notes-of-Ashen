package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// redisConcurrencyLimiter is deliberately narrow so the distributed lease logic
// can be unit-tested without a running Redis server.
type redisConcurrencyLimiter interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// ConcurrencyLimitMiddleware limits long-lived requests (such as SSE) per client
// IP across API instances. Redis failures are fail-closed because allowing an
// unbounded number of model streams during an outage would defeat the safeguard.
type ConcurrencyLimitMiddleware struct {
	redisClient redisConcurrencyLimiter
	name        string
	limit       int64
	ttl         time.Duration
	forwarded   basehandler.ForwardedOptions
}

func NewFailClosedConcurrencyLimitMiddleware(redisClient *redis.Client, name string, limit int64, ttl time.Duration, forwarded ...basehandler.ForwardedOptions) *ConcurrencyLimitMiddleware {
	var client redisConcurrencyLimiter
	if redisClient != nil {
		client = redisClient
	}
	options := basehandler.ForwardedOptions{}
	if len(forwarded) > 0 {
		options = forwarded[0]
	}
	return &ConcurrencyLimitMiddleware{
		redisClient: client,
		name:        name,
		limit:       limit,
		ttl:         ttl,
		forwarded:   options,
	}
}

const concurrencyRedisTimeout = 200 * time.Millisecond

// 并发槽位必须由独立令牌标识，而不是一个单纯计数器：长于 TTL 的旧请求若在
// 计数器过期后结束，会错误地释放后来请求的槽位。ZSET score 是每个租约到期时间，
// 可原子清理过期租约，也让 release 只影响自己的 token。
const acquireConcurrencyScript = `
local now = tonumber(ARGV[1])
local expiresAt = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local token = ARGV[4]
redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", now)
if redis.call("ZCARD", KEYS[1]) >= limit then
  return 0
end
redis.call("ZADD", KEYS[1], expiresAt, token)
redis.call("PEXPIRE", KEYS[1], math.max(expiresAt - now, 1))
return 1
`

const releaseConcurrencyScript = `
local now = tonumber(ARGV[1])
redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", now)
redis.call("ZREM", KEYS[1], ARGV[2])
local latest = redis.call("ZRANGE", KEYS[1], -1, -1, "WITHSCORES")
if #latest == 0 then
  redis.call("DEL", KEYS[1])
else
  redis.call("PEXPIRE", KEYS[1], math.max(tonumber(latest[2]) - now, 1))
end
return 1
`

const renewConcurrencyScript = `
local now = tonumber(ARGV[1])
local expiresAt = tonumber(ARGV[2])
local token = ARGV[3]
redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", now)
if redis.call("ZSCORE", KEYS[1], token) == false then
  return 0
end
redis.call("ZADD", KEYS[1], expiresAt, token)
local latest = redis.call("ZRANGE", KEYS[1], -1, -1, "WITHSCORES")
redis.call("PEXPIRE", KEYS[1], math.max(tonumber(latest[2]) - now, 1))
return 1
`

func (m *ConcurrencyLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.redisClient == nil || m.limit < 1 || m.ttl <= 0 {
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("rag concurrency limiter unavailable"))
			return
		}
		ip := basehandler.Meta(r, m.forwarded).IP
		key := security.RateLimitKey(m.name, ip)
		leaseToken := newConcurrencyLeaseToken()
		acquireCtx, cancel := context.WithTimeout(r.Context(), concurrencyRedisTimeout)
		now := time.Now()
		acquired, err := m.redisClient.Eval(acquireCtx, acquireConcurrencyScript, []string{key}, now.UnixMilli(), now.Add(m.ttl).UnixMilli(), m.limit, leaseToken).Int64()
		cancel()
		if err != nil {
			logx.Errorf("concurrency limit unavailable: name=%s", m.name)
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("rag concurrency limiter unavailable"))
			return
		}
		if acquired != 1 {
			response.ErrorCtx(r.Context(), w, apperrors.TooManyRequests("too many concurrent requests"))
			return
		}

		// Redis 不可用或租约意外丢失时，主动取消下游 context。对于 SSE 这会
		// 中止上游模型请求，保持 fail-closed，而不是在失去全局配额后继续消耗模型。
		streamCtx, cancelStream := context.WithCancel(r.Context())
		stopHeartbeat := m.startHeartbeat(streamCtx, key, leaseToken, cancelStream)
		defer func() {
			stopHeartbeat()
			cancelStream()
			m.release(key, leaseToken)
		}()
		next(w, r.WithContext(streamCtx))
	}
}

func (m *ConcurrencyLimitMiddleware) startHeartbeat(parent context.Context, key, token string, cancelStream context.CancelFunc) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	interval := m.ttl / 3
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-parent.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				if !m.renew(key, token) {
					// 令牌/IP 都不写日志；仅保留限流器名称用于运维定位。
					logx.Errorf("concurrency lease renewal failed: name=%s", m.name)
					cancelStream()
					return
				}
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func (m *ConcurrencyLimitMiddleware) renew(key, token string) bool {
	if m == nil || m.redisClient == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), concurrencyRedisTimeout)
	defer cancel()
	now := time.Now()
	renewed, err := m.redisClient.Eval(ctx, renewConcurrencyScript, []string{key}, now.UnixMilli(), now.Add(m.ttl).UnixMilli(), token).Int64()
	return err == nil && renewed == 1
}

func (m *ConcurrencyLimitMiddleware) release(key, token string) {
	if m == nil || m.redisClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), concurrencyRedisTimeout)
	defer cancel()
	if err := m.redisClient.Eval(ctx, releaseConcurrencyScript, []string{key}, time.Now().UnixMilli(), token).Err(); err != nil {
		// The acquire TTL bounds leaked slots if the client disconnects while Redis
		// is unavailable. Do not expose the key/IP in logs.
		logx.Errorf("concurrency limit release failed: name=%s", m.name)
	}
}

func newConcurrencyLeaseToken() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err == nil {
		return hex.EncodeToString(raw)
	}
	// crypto/rand 故障极罕见；回退值仍包含纳秒时间和单调计数语义所需的高熵
	// 字符串形式。令牌不暴露给客户端，也不作为鉴权凭据。
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}
