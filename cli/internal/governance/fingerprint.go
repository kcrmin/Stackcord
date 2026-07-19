package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProtectedFingerprint binds the governance policy and all protected authored meaning.
func ProtectedFingerprint(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(".harness", "governance.yaml"),
		filepath.Join("specs", "product"),
		filepath.Join("specs", "policies"),
		"contracts",
	}
	paths := []string{}
	for _, relative := range candidates {
		base := filepath.Join(root, relative)
		info, statErr := os.Lstat(base)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("protected governance source cannot be a symlink: %s", filepath.ToSlash(relative))
		}
		if info.Mode().IsRegular() {
			paths = append(paths, base)
			continue
		}
		if !info.IsDir() {
			return "", fmt.Errorf("protected governance source must be a regular file or directory: %s", filepath.ToSlash(relative))
		}
		if err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("protected governance source cannot contain symlinks: %s", filepath.ToSlash(path))
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("protected governance source must contain only regular files: %s", filepath.ToSlash(path))
			}
			paths = append(paths, path)
			return nil
		}); err != nil {
			return "", err
		}
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("protected governance source escaped repository")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(filepath.ToSlash(relative)))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
