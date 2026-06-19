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
	evalValue int64
	evalErr   error
	evalCalls int
	incrCalls int
}

func (f *fakeRedisLimiter) Eval(context.Context, string, []string, ...interface{}) *redis.Cmd {
	f.evalCalls++
	return redis.NewCmdResult(f.evalValue, f.evalErr)
}

func TestRateLimitMiddlewareAllowsWhenRedisClientIsNil(t *testing.T) {
	middleware := NewRateLimitMiddleware(nil, "login", 1, time.Minute)

	rec := serveRateLimitedRequest(middleware)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestRateLimitMiddlewareAllowsWhenRedisEvalFails(t *testing.T) {
	middleware := &RateLimitMiddleware{
		redisClient: &fakeRedisLimiter{evalErr: errors.New("redis unavailable")},
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
		redisClient: &fakeRedisLimiter{evalValue: 2},
		name:        "login",
		limit:       1,
		window:      time.Minute,
	}

	rec := serveRateLimitedRequest(middleware)

	assertErrorResponse(t, rec, http.StatusTooManyRequests, 42900)
}

func TestRateLimitMiddlewareUsesAtomicEval(t *testing.T) {
	client := &fakeRedisLimiter{evalValue: 1}
	middleware := &RateLimitMiddleware{
		redisClient: client,
		name:        "login",
		limit:       1,
		window:      time.Minute,
	}

	rec := serveRateLimitedRequest(middleware)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if client.evalCalls != 1 {
		t.Fatalf("Eval calls = %d, want 1", client.evalCalls)
	}
	if client.incrCalls != 0 {
		t.Fatalf("Incr calls = %d, want 0", client.incrCalls)
	}
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
