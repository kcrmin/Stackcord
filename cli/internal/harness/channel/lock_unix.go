//go:build !windows

package channel

import (
	"fmt"
	"os"
	"syscall"
)

func fileLock(p string) (func(), error) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("channel is busy: %w", e)
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
func secureDirectory(p string) error { return os.Chmod(p, 0700) }
