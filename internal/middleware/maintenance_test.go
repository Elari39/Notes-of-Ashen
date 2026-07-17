package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
)

func TestMaintenanceBlocksConcurrentRestoreBeforeBodyParsing(t *testing.T) {
	if !security.TryStartRestore() {
		t.Fatal("could not start test restore state")
	}
	defer security.EndRestore()

	middleware := NewMaintenanceMiddleware(nil)
	called := false
	handler := middleware.Handle(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups/restore", nil)

	handler(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if called {
		t.Fatal("restore handler was called while maintenance was active")
	}
}

func TestMaintenanceRedisCheckTimesOutAndContinues(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:                  "maintenance-test.invalid:6379",
		ContextTimeoutEnabled: true,
		MaxRetries:            -1,
		Dialer: func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	defer client.Close()

	middleware := NewMaintenanceMiddleware(client)
	called := false
	handler := middleware.Handle(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)

	startedAt := time.Now()
	handler(recorder, req)
	elapsed := time.Since(startedAt)

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("Redis 检查超时后未继续后续处理：called=%t status=%d", called, recorder.Code)
	}
	if elapsed > maintenanceRedisTimeout+300*time.Millisecond {
		t.Fatalf("Redis 检查耗时 %s，期望不超过 %s", elapsed, maintenanceRedisTimeout+300*time.Millisecond)
	}
}
