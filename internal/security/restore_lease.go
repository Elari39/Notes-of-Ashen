package security

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	restoreLeaseTTL              = 30 * time.Minute
	restoreLeaseRenewInterval    = restoreLeaseTTL / 3
	restoreLeaseOperationTimeout = 5 * time.Second
)

const renewRestoreLeaseScript = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call("PEXPIRE", KEYS[1], ARGV[2])
return 1
`

const releaseRestoreLeaseScript = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call("DEL", KEYS[1])
return 1
`

const checkRestoreLeaseScript = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
return 1
`

var (
	ErrRestoreLeaseHeld = errors.New("restore lease is already held")
	ErrRestoreLeaseLost = errors.New("restore lease ownership was lost")
)

type restoreLeaseRedis interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// RestoreLease keeps the distributed maintenance lock alive while a destructive
// full-site restore is running. The owner token prevents an expired worker from
// deleting or renewing a newer worker's lock.
type RestoreLease struct {
	redis   restoreLeaseRedis
	owner   string
	ttl     time.Duration
	refresh time.Duration
	ctx     context.Context
	cancel  context.CancelFunc

	stop chan struct{}
	done chan struct{}
	lost atomic.Bool

	releaseOnce sync.Once
	releaseErr  error
}

func AcquireRestoreLease(ctx context.Context, client restoreLeaseRedis) (*RestoreLease, context.Context, error) {
	return acquireRestoreLease(ctx, client, restoreLeaseTTL, restoreLeaseRenewInterval)
}

func acquireRestoreLease(ctx context.Context, client restoreLeaseRedis, ttl, refresh time.Duration) (*RestoreLease, context.Context, error) {
	if client == nil {
		return nil, nil, errors.New("restore lease redis client is nil")
	}
	if ttl <= 0 {
		return nil, nil, errors.New("restore lease ttl is invalid")
	}
	if refresh <= 0 || refresh >= ttl {
		refresh = ttl / 3
	}
	owner, err := RandomID()
	if err != nil {
		return nil, nil, err
	}
	locked, err := client.SetNX(ctx, RestoreMaintenanceKey, owner, ttl).Result()
	if err != nil {
		return nil, nil, err
	}
	if !locked {
		return nil, nil, ErrRestoreLeaseHeld
	}
	restoreCtx, cancel := context.WithCancel(ctx)
	lease := &RestoreLease{
		redis: client, owner: owner, ttl: ttl, refresh: refresh, ctx: restoreCtx, cancel: cancel,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go lease.renewLoop()
	return lease, restoreCtx, nil
}

func (l *RestoreLease) renewLoop() {
	defer close(l.done)
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(context.Background(), restoreLeaseOperationTimeout)
			result, err := l.redis.Eval(
				renewCtx,
				renewRestoreLeaseScript,
				[]string{RestoreMaintenanceKey},
				l.owner,
				l.ttl.Milliseconds(),
			).Int64()
			cancel()
			if err != nil || result != 1 {
				l.markLost()
				return
			}
		}
	}
}

func (l *RestoreLease) markLost() {
	l.lost.Store(true)
	l.cancel()
}

// CheckLocal observes local cancellation and the renewal loop without adding a
// Redis round trip for every individual media file.
func (l *RestoreLease) CheckLocal() error {
	if l == nil || l.lost.Load() {
		return ErrRestoreLeaseLost
	}
	select {
	case <-l.ctx.Done():
		if l.lost.Load() {
			return ErrRestoreLeaseLost
		}
		return l.ctx.Err()
	default:
		return nil
	}
}

// Check verifies ownership synchronously before a destructive stage. This
// closes the interval between periodic renewals and a cross-instance takeover.
func (l *RestoreLease) Check() error {
	if err := l.CheckLocal(); err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(context.Background(), restoreLeaseOperationTimeout)
	defer cancel()
	result, err := l.redis.Eval(
		checkCtx,
		checkRestoreLeaseScript,
		[]string{RestoreMaintenanceKey},
		l.owner,
	).Int64()
	if err != nil || result != 1 {
		l.markLost()
		return ErrRestoreLeaseLost
	}
	return nil
}

// Release stops renewal and only deletes the key when this lease still owns it.
func (l *RestoreLease) Release() error {
	if l == nil {
		return nil
	}
	l.releaseOnce.Do(func() {
		close(l.stop)
		l.cancel()
		select {
		case <-l.done:
		case <-time.After(restoreLeaseOperationTimeout):
			l.releaseErr = errors.New("restore lease renewal did not stop")
			return
		}
		releaseCtx, cancel := context.WithTimeout(context.Background(), restoreLeaseOperationTimeout)
		defer cancel()
		_, l.releaseErr = l.redis.Eval(
			releaseCtx,
			releaseRestoreLeaseScript,
			[]string{RestoreMaintenanceKey},
			l.owner,
		).Int64()
	})
	return l.releaseErr
}
