package security

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRestoreLeaseRedis struct {
	mu         sync.Mutex
	owner      string
	renewCalls int
}

func (f *fakeRestoreLeaseRedis) SetNX(_ context.Context, _ string, value interface{}, _ time.Duration) *redis.BoolCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.owner != "" {
		return redis.NewBoolResult(false, nil)
	}
	owner, _ := value.(string)
	f.owner = owner
	return redis.NewBoolResult(true, nil)
}

func (f *fakeRestoreLeaseRedis) Eval(_ context.Context, script string, _ []string, args ...interface{}) *redis.Cmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	owner, _ := args[0].(string)
	if f.owner != owner {
		return redis.NewCmdResult(int64(0), nil)
	}
	switch script {
	case renewRestoreLeaseScript:
		f.renewCalls++
		return redis.NewCmdResult(int64(1), nil)
	case checkRestoreLeaseScript:
		return redis.NewCmdResult(int64(1), nil)
	case releaseRestoreLeaseScript:
		f.owner = ""
		return redis.NewCmdResult(int64(1), nil)
	default:
		return redis.NewCmdResult(nil, errors.New("unexpected script"))
	}
}

func (f *fakeRestoreLeaseRedis) setOwner(owner string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.owner = owner
}

func (f *fakeRestoreLeaseRedis) snapshot() (string, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.owner, f.renewCalls
}

func TestRestoreLeaseRenewsAndReleasesOwnedLock(t *testing.T) {
	client := &fakeRestoreLeaseRedis{}
	lease, _, err := acquireRestoreLease(context.Background(), client, time.Second, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("acquireRestoreLease() error = %v", err)
	}
	defer lease.Release()

	deadline := time.Now().Add(time.Second)
	for {
		_, renewCalls := client.snapshot()
		if renewCalls > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("restore lease was not renewed")
		}
		time.Sleep(time.Millisecond)
	}
	if err := lease.Check(); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if owner, _ := client.snapshot(); owner != "" {
		t.Fatalf("lock owner = %q, want empty", owner)
	}
}

func TestRestoreLeaseDoesNotReleaseAnotherOwnerAndCancelsWhenLost(t *testing.T) {
	client := &fakeRestoreLeaseRedis{}
	lease, restoreCtx, err := acquireRestoreLease(context.Background(), client, time.Second, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("acquireRestoreLease() error = %v", err)
	}
	client.setOwner("new-owner")

	select {
	case <-restoreCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("restore context was not canceled after lease ownership was lost")
	}
	if !errors.Is(lease.Check(), ErrRestoreLeaseLost) {
		t.Fatalf("Check() error = %v, want ErrRestoreLeaseLost", lease.Check())
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if owner, _ := client.snapshot(); owner != "new-owner" {
		t.Fatalf("lock owner = %q, want new-owner", owner)
	}
}

func TestRestoreLeaseCheckDetectsTakeoverBeforeNextRenewal(t *testing.T) {
	client := &fakeRestoreLeaseRedis{}
	lease, _, err := acquireRestoreLease(context.Background(), client, time.Hour, time.Hour/3)
	if err != nil {
		t.Fatalf("acquireRestoreLease() error = %v", err)
	}
	defer lease.Release()
	client.setOwner("new-owner")

	if !errors.Is(lease.Check(), ErrRestoreLeaseLost) {
		t.Fatalf("Check() error = %v, want ErrRestoreLeaseLost", lease.Check())
	}
}

func TestAcquireRestoreLeaseRejectsExistingOwner(t *testing.T) {
	client := &fakeRestoreLeaseRedis{owner: "other-owner"}
	_, _, err := acquireRestoreLease(context.Background(), client, time.Second, time.Millisecond)
	if !errors.Is(err, ErrRestoreLeaseHeld) {
		t.Fatalf("acquireRestoreLease() error = %v, want ErrRestoreLeaseHeld", err)
	}
}
