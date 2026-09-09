package pathresolve

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var finalPath = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

// Resolve opens the actual target and asks Windows for its canonical path.
// Unlike a component-by-component walk, this works when a sandbox allows the
// target directory but not enumeration of its user-profile ancestors. Reparse
// points are still resolved by the kernel; no lexical fallback is accepted.
func Resolve(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	name, err := syscall.UTF16PtrFromString(absolute)
	if err != nil {
		return "", err
	}
	handle, err := syscall.CreateFile(name, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)
	size := uint32(512)
	for size <= 65536 {
		buffer := make([]uint16, size)
		n, _, callErr := finalPath.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), uintptr(size), 0)
		if n == 0 {
			return "", fmt.Errorf("resolve Windows target: %w", callErr)
		}
		if n >= uintptr(size) {
			size = uint32(n) + 1
			continue
		}
		resolved := syscall.UTF16ToString(buffer[:n])
		if strings.HasPrefix(resolved, `\\?\UNC\`) {
			resolved = `\\` + strings.TrimPrefix(resolved, `\\?\UNC\`)
		} else {
			resolved = strings.TrimPrefix(resolved, `\\?\`)
		}
		if !filepath.IsAbs(resolved) {
			return "", fmt.Errorf("Windows returned a non-absolute target")
		}
		return filepath.Clean(resolved), nil
	}
	return "", fmt.Errorf("resolved Windows path exceeds supported length")
}
