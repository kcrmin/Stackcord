//go:build !windows

// Package pathresolve resolves an existing path through the operating system.
package pathresolve

import "path/filepath"

func Resolve(path string) (string, error) { return filepath.EvalSymlinks(path) }
