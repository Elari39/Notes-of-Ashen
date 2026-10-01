//go:build !windows && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package media

import (
	"errors"
	"os"
)

func tryMediaFileLock(_ *os.File) (bool, error) {
	return false, errors.New("media operation locking is unsupported on this platform")
}
