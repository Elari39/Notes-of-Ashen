package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// lockMediaOperations serializes upload/delete transactions and their recovery
// across all processes sharing a media directory. The OS releases the lock on
// process exit, so recovery never relies on a guessed age or an expiring lease.
// Keep the lock file in place: unlinking it could give waiters different inodes.
func lockMediaOperations(ctx context.Context, root string) (func(), error) {
	file, err := os.OpenFile(filepath.Join(root, ".operations.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open media operation lock: %w", err)
	}
	closeFile := func() {
		if err := file.Close(); err != nil {
			logx.Errorf("close media operation lock: %v", err)
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			closeFile()
			return nil, err
		}
		locked, err := tryMediaFileLock(file)
		if err != nil {
			closeFile()
			return nil, fmt.Errorf("lock media operations: %w", err)
		}
		if locked {
			return sync.OnceFunc(closeFile), nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			closeFile()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
