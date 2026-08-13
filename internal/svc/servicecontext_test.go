package svc

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"notes-of-ashen/model"
)

func TestRedisStartupPingFailureAuthenticationIsExplicitAndRedacted(t *testing.T) {
	const secret = "redis-password-must-not-appear"
	err := errors.New("WRONGPASS invalid username-password pair: " + secret)

	got := redisStartupPingFailure(err)
	if !strings.Contains(got.Error(), "redis PING authentication failed") {
		t.Fatalf("redisStartupPingFailure() = %q, want explicit authentication failure", got)
	}
	if strings.Contains(got.Error(), secret) || strings.Contains(got.Error(), "WRONGPASS") {
		t.Fatalf("redisStartupPingFailure() leaked Redis error detail: %q", got)
	}
}

func TestRedisStartupPingFailurePreservesNonAuthenticationError(t *testing.T) {
	err := errors.New("dial tcp redis:6379: connect: connection refused")

	got := redisStartupPingFailure(err)
	if !strings.Contains(got.Error(), "redis PING failed") {
		t.Fatalf("redisStartupPingFailure() = %q, want PING failure context", got)
	}
	if !errors.Is(got, err) {
		t.Fatalf("redisStartupPingFailure() = %v, want wrapped original error", got)
	}
}

func TestIsRedisAuthenticationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "wrong password", err: errors.New("WRONGPASS invalid username-password pair or user is disabled."), want: true},
		{name: "missing authentication", err: errors.New("NOAUTH Authentication required."), want: true},
		{name: "password configured against unauthenticated redis", err: errors.New("ERR AUTH <password> called without any password configured for the default user"), want: true},
		{name: "network failure", err: errors.New("dial tcp redis:6379: connect: connection refused"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRedisAuthenticationError(tt.err); got != tt.want {
				t.Fatalf("isRedisAuthenticationError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestInitializeSearchWithEnsureReturnsAfterInitialSuccess(t *testing.T) {
	var calls atomic.Int32
	cancel := initializeSearchWithEnsure(func(context.Context) error {
		calls.Add(1)
		return nil
	}, time.Millisecond)

	if cancel != nil {
		t.Fatal("initial success should not start a retry goroutine")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("ensure calls = %d, want 1", got)
	}
}

func TestInitializeSearchWithEnsureRetriesUntilSuccess(t *testing.T) {
	var calls atomic.Int32
	recovered := make(chan struct{})
	cancel := initializeSearchWithEnsure(func(context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("temporarily unavailable")
		}
		close(recovered)
		return nil
	}, time.Millisecond)
	if cancel == nil {
		t.Fatal("initial failure should start a retry goroutine")
	}
	defer cancel()

	select {
	case <-recovered:
	case <-time.After(time.Second):
		t.Fatal("search initialization did not recover after retry")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("ensure calls = %d, want 2", got)
	}
}

func TestInitializeSearchWithEnsureCancelStopsInFlightRetry(t *testing.T) {
	var calls atomic.Int32
	retryStarted := make(chan struct{})
	retryStopped := make(chan struct{})
	cancel := initializeSearchWithEnsure(func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("temporarily unavailable")
		}
		close(retryStarted)
		<-ctx.Done()
		close(retryStopped)
		return ctx.Err()
	}, time.Millisecond)
	if cancel == nil {
		t.Fatal("initial failure should start a retry goroutine")
	}

	select {
	case <-retryStarted:
	case <-time.After(time.Second):
		t.Fatal("background retry did not start")
	}
	cancel()
	select {
	case <-retryStopped:
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop the in-flight retry")
	}
}

func TestStartRefreshTokenCleanupRunsImmediatelyAndCanCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta(`
DELETE FROM refresh_tokens
WHERE expires_at <= ?
   OR (revoked_at IS NOT NULL AND revoked_at <= ?)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	cancel := startRefreshTokenCleanup(model.NewStore(db))
	if cancel == nil {
		t.Fatal("startRefreshTokenCleanup() returned nil cancel")
	}
	cancel()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("cleanup did not run at startup: %v", err)
	}
}

func TestStartRAGChatCleanupRunsImmediatelyAndCanCancel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM rag_chat_sessions WHERE expires_at IS NOT NULL AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	cancel := startRAGChatCleanup(model.NewStore(db))
	if cancel == nil {
		t.Fatal("startRAGChatCleanup() returned nil cancel")
	}
	cancel()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("cleanup did not run at startup: %v", err)
	}
}
