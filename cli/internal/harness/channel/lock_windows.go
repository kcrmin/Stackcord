package channel

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"unsafe"
)

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")
var unlockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("UnlockFileEx")

func fileLock(p string) (func(), error) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	var ov syscall.Overlapped
	r, _, e := lockFileEx.Call(f.Fd(), 3, 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
	if r == 0 {
		f.Close()
		return nil, fmt.Errorf("channel is busy: %w", e)
	}
	return func() { unlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ov))); f.Close() }, nil
}
func secureDirectory(p string) error {
	u, e := user.Current()
	if e != nil {
		return e
	}
	out, e := exec.Command("icacls", p, "/inheritance:r", "/grant:r", "*"+u.Uid+":(OI)(CI)F").CombinedOutput()
	if e != nil {
		return fmt.Errorf("secure local channel permissions: %w: %s", e, out)
	}
	return nil
}
