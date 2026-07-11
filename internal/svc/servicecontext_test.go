package svc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

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
