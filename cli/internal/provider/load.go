package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fullstack-orchestrator/cli/internal/schema"
)

// LoadMapping strictly decodes one stable provider mapping.
func LoadMapping(path string) (Mapping, error) {
	if err := regularProviderFile(path); err != nil {
		return Mapping{}, err
	}
	value, err := schema.LoadYAML[Mapping](path)
	if err != nil {
		return Mapping{}, err
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Mapping{}, err
	}
	if issues := schema.Validate("provider-mapping", raw); len(issues) > 0 {
		return Mapping{}, fmt.Errorf("validate provider mapping: %s", issues[0].Message)
	}
	return value, nil
}

// LoadSnapshot strictly decodes one local normalized provider observation.
func LoadSnapshot(path string) (Snapshot, error) {
	if err := regularProviderFile(path); err != nil {
		return Snapshot{}, err
	}
	value, err := schema.LoadYAML[Snapshot](path)
	if err != nil {
		return Snapshot{}, err
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Snapshot{}, err
	}
	if issues := schema.Validate("provider-snapshot", raw); len(issues) > 0 {
		return Snapshot{}, fmt.Errorf("validate provider snapshot: %s", issues[0].Message)
	}
	return value, nil
}

// ValidateSnapshotLocation prevents provider observations from becoming canonical files.
func ValidateSnapshotLocation(root, path string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	localRoot := filepath.Join(root, ".harness", "local", "providers")
	temporaryRoot, tempErr := filepath.EvalSymlinks(os.TempDir())
	if tempErr != nil {
		temporaryRoot = os.TempDir()
	}
	resolved, resolveErr := filepath.EvalSymlinks(path)
	if resolveErr != nil {
		return resolveErr
	}
	if pathWithin(localRoot, resolved) || pathWithin(temporaryRoot, resolved) {
		return nil
	}
	return fmt.Errorf("provider snapshots must stay under .harness/local/providers or an explicit temporary path")
}

func regularProviderFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("provider input must be a regular non-symlink file: %s", path)
	}
	return nil
}

func pathWithin(parent, child string) bool {
	parent, _ = filepath.Abs(parent)
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
