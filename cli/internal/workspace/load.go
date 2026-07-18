package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/schema"
)

// Load reads and validates committed project and workspace identities.
func Load(root string) (Manifest, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	projectPath := filepath.Join(root, ".harness", "manifest.yaml")
	if err := requireRegularFile(projectPath); err != nil {
		return Manifest{}, err
	}
	project, err := schema.LoadYAML[map[string]any](projectPath)
	if err != nil {
		return Manifest{}, err
	}
	projectID, _ := project["id"].(string)
	if projectID == "" {
		return Manifest{}, fmt.Errorf("project manifest has no stable ID")
	}
	path := filepath.Join(root, ".harness", "workspaces.yaml")
	if err := requireRegularFile(path); err != nil {
		return Manifest{}, err
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Manifest{}, err
	}
	if issues := schema.Validate("workspaces", raw); len(issues) > 0 {
		return Manifest{}, fmt.Errorf("validate %s: %s", path, issues[0].Message)
	}
	manifest, err := schema.LoadYAML[Manifest](path)
	if err != nil {
		return Manifest{}, err
	}
	if manifest.ProjectID != projectID {
		return Manifest{}, fmt.Errorf("workspace project ID %q differs from project manifest %q", manifest.ProjectID, projectID)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	sort.Slice(manifest.Workspaces, func(left, right int) bool {
		return manifest.Workspaces[left].ID < manifest.Workspaces[right].ID
	})
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	ids := make(map[string]struct{}, len(manifest.Workspaces))
	paths := make(map[string]struct{}, len(manifest.Workspaces))
	hasRoot := false
	for _, entry := range manifest.Workspaces {
		if _, exists := ids[entry.ID]; exists {
			return fmt.Errorf("duplicate workspace ID %q", entry.ID)
		}
		ids[entry.ID] = struct{}{}
		clean, err := safeRelative(entry.Path)
		if err != nil {
			return fmt.Errorf("workspace %s path: %w", entry.ID, err)
		}
		if _, exists := paths[clean]; exists {
			return fmt.Errorf("duplicate workspace path %q", clean)
		}
		paths[clean] = struct{}{}
		if entry.Kind == "root" {
			if clean != "." {
				return fmt.Errorf("root workspace %s must use path .", entry.ID)
			}
			if hasRoot {
				return fmt.Errorf("workspace manifest has more than one root")
			}
			hasRoot = true
		}
		if entry.CommandsPath != "" {
			if _, err := safeRelative(entry.CommandsPath); err != nil {
				return fmt.Errorf("workspace %s commands path: %w", entry.ID, err)
			}
		}
	}
	if !hasRoot {
		return fmt.Errorf("workspace manifest has no root workspace")
	}
	for _, entry := range manifest.Workspaces {
		for _, dependency := range entry.Dependencies {
			if _, exists := ids[dependency]; !exists {
				return fmt.Errorf("workspace %s references missing dependency %s", entry.ID, dependency)
			}
		}
	}
	return nil
}

func safeRelative(value string) (string, error) {
	if value == "" || filepath.IsAbs(value) {
		return "", fmt.Errorf("must be a non-empty relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("must stay inside its repository")
	}
	return filepath.ToSlash(clean), nil
}

func loadBridge(path string) (Bridge, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Bridge{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Bridge{}, fmt.Errorf("workspace bridge must be a regular non-symlink file: %s", path)
	}
	bridge, err := schema.LoadYAML[Bridge](path)
	if err != nil {
		return Bridge{}, err
	}
	if bridge.SchemaVersion != 1 || bridge.ProjectID == "" || bridge.RootRemote == "" || bridge.WorkspaceID == "" || bridge.Discovery == "" {
		return Bridge{}, fmt.Errorf("workspace bridge identity is incomplete")
	}
	switch bridge.Discovery {
	case "git-superproject", "root-remote", "configured-path":
	default:
		return Bridge{}, fmt.Errorf("workspace bridge discovery method %q is unsupported", bridge.Discovery)
	}
	if _, err := safeRelative(bridge.CommandsPath); err != nil {
		return Bridge{}, fmt.Errorf("workspace bridge commands path: %w", err)
	}
	return bridge, nil
}

func requireRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("canonical workspace file must be a regular non-symlink file: %s", path)
	}
	return nil
}
