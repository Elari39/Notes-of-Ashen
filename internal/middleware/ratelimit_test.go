package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisLimiter struct {
	incrValue int64
	incrErr   error
	expireErr error
}

func (f fakeRedisLimiter) Incr(context.Context, string) *redis.IntCmd {
	return redis.NewIntResult(f.incrValue, f.incrErr)
}

func (f fakeRedisLimiter) Expire(context.Context, string, time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(f.expireErr == nil, f.expireErr)
}

func TestRateLimitMiddlewareAllowsWhenRedisClientIsNil(t *testing.T) {
	middleware := NewRateLimitMiddleware(nil, "login", 1, time.Minute)

	rec := serveRateLimitedRequest(middleware)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestRateLimitMiddlewareAllowsWhenRedisIncrFails(t *testing.T) {
	middleware := &RateLimitMiddleware{
		redisClient: fakeRedisLimiter{incrErr: errors.New("redis unavailable")},
		name:        "login",
		limit:       1,
		window:      time.Minute,
	}

	rec := serveRateLimitedRequest(middleware)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestRateLimitMiddlewareAllowsWhenRedisExpireFails(t *testing.T) {
	middleware := &RateLimitMiddleware{
		redisClient: fakeRedisLimiter{incrValue: 1, expireErr: errors.New("redis unavailable")},
		name:        "login",
		limit:       1,
		window:      time.Minute,
	}

	rec := serveRateLimitedRequest(middleware)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestRateLimitMiddlewareRejectsOverLimit(t *testing.T) {
	middleware := &RateLimitMiddleware{
		redisClient: fakeRedisLimiter{incrValue: 2},
		name:        "login",
		limit:       1,
		window:      time.Minute,
	}

	rec := serveRateLimitedRequest(middleware)

	assertErrorResponse(t, rec, http.StatusTooManyRequests, 42900)
}

func serveRateLimitedRequest(middleware *RateLimitMiddleware) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	middleware.Handle(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).ServeHTTP(rec, req)
	return rec
}
