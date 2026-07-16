package context

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot returns the nearest ancestor containing .harness/manifest.yaml.
func FindRoot(start string) (string, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start path: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect start path: %w", err)
	}
	if !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}
	realStart, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve start symlinks: %w", err)
	}
	volume := filepath.VolumeName(realStart)
	for current := realStart; ; current = filepath.Dir(current) {
		if filepath.VolumeName(current) != volume {
			break
		}
		manifest := filepath.Join(current, ".harness", "manifest.yaml")
		if stat, statErr := os.Stat(manifest); statErr == nil && !stat.IsDir() {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", fmt.Errorf("project root not found from %s", start)
}
