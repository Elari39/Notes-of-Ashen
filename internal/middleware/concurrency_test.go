package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type concurrencyEvalCall struct {
	script string
	keys   []string
	args   []interface{}
}

type fakeConcurrencyLimiter struct {
	mu      sync.Mutex
	calls   []concurrencyEvalCall
	acquire int64
	renew   int64
	release int64
	err     error
}

func (f *fakeConcurrencyLimiter) Eval(_ context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	f.mu.Lock()
	f.calls = append(f.calls, concurrencyEvalCall{script: script, keys: append([]string(nil), keys...), args: append([]interface{}(nil), args...)})
	f.mu.Unlock()
	if f.err != nil {
		return redis.NewCmdResult(int64(0), f.err)
	}
	switch script {
	case acquireConcurrencyScript:
		return redis.NewCmdResult(f.acquire, nil)
	case renewConcurrencyScript:
		return redis.NewCmdResult(f.renew, nil)
	case releaseConcurrencyScript:
		return redis.NewCmdResult(f.release, nil)
	default:
		return redis.NewCmdResult(int64(0), errors.New("unexpected script"))
	}
}

func (f *fakeConcurrencyLimiter) snapshot() []concurrencyEvalCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]concurrencyEvalCall(nil), f.calls...)
}

func TestConcurrencyLimitFailsClosedWhenRedisUnavailable(t *testing.T) {
	middleware := &ConcurrencyLimitMiddleware{
		redisClient: &fakeConcurrencyLimiter{err: errors.New("redis down")},
		name:        "rag_chat_concurrent",
		limit:       2,
		ttl:         time.Minute,
	}
	rec := serveConcurrencyLimitedRequest(middleware, func(http.ResponseWriter, *http.Request) {})
	assertErrorResponse(t, rec, http.StatusServiceUnavailable, 50300)
}

func TestConcurrencyLimitRejectsWhenLeaseCannotBeAcquired(t *testing.T) {
	middleware := &ConcurrencyLimitMiddleware{
		redisClient: &fakeConcurrencyLimiter{acquire: 0},
		name:        "rag_chat_concurrent",
		limit:       2,
		ttl:         time.Minute,
	}
	rec := serveConcurrencyLimitedRequest(middleware, func(http.ResponseWriter, *http.Request) {})
	assertErrorResponse(t, rec, http.StatusTooManyRequests, 42900)
}

func TestConcurrencyLimitReleasesOnlyItsOwnLeaseToken(t *testing.T) {
	client := &fakeConcurrencyLimiter{acquire: 1, renew: 1, release: 1}
	middleware := &ConcurrencyLimitMiddleware{
		redisClient: client,
		name:        "rag_chat_concurrent",
		limit:       2,
		ttl:         time.Hour,
	}
	rec := serveConcurrencyLimitedRequest(middleware, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	calls := client.snapshot()
	if len(calls) != 2 || calls[0].script != acquireConcurrencyScript || calls[1].script != releaseConcurrencyScript {
		t.Fatalf("calls = %#v, want acquire then release", calls)
	}
	if got, want := calls[0].args[3], calls[1].args[1]; got != want {
		t.Fatalf("release token = %#v, want acquired token %#v", got, want)
	}
	if token, _ := calls[0].args[3].(string); token == "" || strings.Contains(token, "127.0.0.1") {
		t.Fatalf("unsafe lease token = %q", token)
	}
}

func TestConcurrencyLimitRenewsLongLivedLease(t *testing.T) {
	client := &fakeConcurrencyLimiter{acquire: 1, renew: 1, release: 1}
	middleware := &ConcurrencyLimitMiddleware{
		redisClient: client,
		name:        "rag_chat_concurrent",
		limit:       2,
		ttl:         9 * time.Millisecond,
	}
	rec := serveConcurrencyLimitedRequest(middleware, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(16 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	var renews int
	for _, call := range client.snapshot() {
		if call.script == renewConcurrencyScript {
			renews++
		}
	}
	if renews == 0 {
		t.Fatal("long-lived request did not renew its concurrency lease")
	}
}

func TestConcurrencyLimitCancelsRequestWhenRenewalFails(t *testing.T) {
	client := &fakeConcurrencyLimiter{acquire: 1, renew: 0, release: 1}
	middleware := &ConcurrencyLimitMiddleware{
		redisClient: client,
		name:        "rag_chat_concurrent",
		limit:       2,
		ttl:         9 * time.Millisecond,
	}
	ctxCancelled := make(chan struct{})
	rec := serveConcurrencyLimitedRequest(middleware, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			close(ctxCancelled)
			w.WriteHeader(http.StatusNoContent)
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	select {
	case <-ctxCancelled:
	default:
		t.Fatal("request context was not cancelled after lease renewal failure")
	}
}

func serveConcurrencyLimitedRequest(middleware *ConcurrencyLimitMiddleware, next http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rag/chat/stream", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	middleware.Handle(next).ServeHTTP(rec, req.WithContext(context.Background()))
	return rec
}
