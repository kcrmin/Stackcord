package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindRoot locates service-wide context, preferring actual Git superproject evidence.
func FindRoot(ctx context.Context, start string) (Root, error) {
	directory, err := realDirectory(start)
	if err != nil {
		return Root{}, err
	}
	gitTop, _ := gitRead(ctx, directory, "rev-parse", "--show-toplevel")
	superproject, _ := gitRead(ctx, directory, "rev-parse", "--show-superproject-working-tree")
	if superproject != "" {
		if rootPath, found, findErr := findProjectAncestor(superproject); findErr != nil {
			return Root{}, findErr
		} else if found {
			return loadLocatedRoot(rootPath, gitTop, directory, RootFromSuperproject)
		}
	}
	if rootPath, found, findErr := findProjectAncestor(directory); findErr != nil {
		return Root{}, findErr
	} else if found {
		return loadLocatedRoot(rootPath, gitTop, directory, RootFromAncestor)
	}
	if bridgePath, found, findErr := findBridgeAncestor(directory); findErr != nil {
		return Root{}, findErr
	} else if found {
		bridge, loadErr := loadBridge(bridgePath)
		if loadErr != nil {
			return Root{}, loadErr
		}
		return Root{}, &IncompleteContextError{ProjectID: bridge.ProjectID, WorkspaceID: bridge.WorkspaceID, RootRemote: bridge.RootRemote}
	}
	return Root{}, fmt.Errorf("orchestration root not found from %s", start)
}

func loadLocatedRoot(rootPath, gitTop, startDirectory string, source RootSource) (Root, error) {
	manifest, err := Load(rootPath)
	if err != nil {
		return Root{}, err
	}
	current := startDirectory
	if gitTop != "" && !samePath(gitTop, rootPath) {
		current = gitTop
	}
	workspaceID, entry, err := currentEntry(rootPath, current, manifest)
	if err != nil {
		return Root{}, err
	}
	if gitTop != "" && !samePath(gitTop, rootPath) {
		bridgePath := filepath.Join(gitTop, ".harness", "bridge.yaml")
		if _, statErr := os.Lstat(bridgePath); statErr == nil {
			bridge, bridgeErr := loadBridge(bridgePath)
			if bridgeErr != nil {
				return Root{}, bridgeErr
			}
			if bridgeErr = validateBridge(manifest, entry, bridge); bridgeErr != nil {
				return Root{}, bridgeErr
			}
		} else if !os.IsNotExist(statErr) {
			return Root{}, statErr
		}
	}
	return Root{Path: rootPath, CurrentWorkspaceID: workspaceID, Source: source, Manifest: manifest}, nil
}

func currentEntry(rootPath, current string, manifest Manifest) (string, Entry, error) {
	var selected Entry
	selectedLength := -1
	for _, entry := range manifest.Workspaces {
		candidate := filepath.Join(rootPath, filepath.FromSlash(entry.Path))
		candidate, _ = filepath.Abs(candidate)
		if samePath(current, candidate) || pathInside(candidate, current) {
			if len(candidate) > selectedLength {
				selected = entry
				selectedLength = len(candidate)
			}
		}
	}
	if selectedLength < 0 {
		return "", Entry{}, fmt.Errorf("current repository %s is not declared in workspaces", current)
	}
	return selected.ID, selected, nil
}

func validateBridge(manifest Manifest, entry Entry, bridge Bridge) error {
	if bridge.ProjectID != manifest.ProjectID {
		return fmt.Errorf("child bridge project ID %q differs from root %q", bridge.ProjectID, manifest.ProjectID)
	}
	if bridge.WorkspaceID != entry.ID {
		return fmt.Errorf("child bridge workspace ID %q differs from root entry %q", bridge.WorkspaceID, entry.ID)
	}
	if manifest.RootRemote != "" && bridge.RootRemote != manifest.RootRemote {
		return fmt.Errorf("child bridge root remote differs from canonical root remote")
	}
	if entry.ContractFingerprint != "" && bridge.ContractFingerprint != entry.ContractFingerprint {
		return fmt.Errorf("child bridge contract fingerprint differs from canonical workspace")
	}
	if entry.CommandsPath != "" && bridge.CommandsPath != entry.CommandsPath {
		return fmt.Errorf("child bridge commands path differs from canonical workspace")
	}
	return nil
}

func realDirectory(start string) (string, error) {
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
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve start symlinks: %w", err)
	}
	return resolved, nil
}

func findProjectAncestor(start string) (string, bool, error) {
	path, found, err := findHarnessFile(start, "manifest.yaml")
	if err != nil || !found {
		return "", found, err
	}
	return filepath.Dir(filepath.Dir(path)), true, nil
}

func findBridgeAncestor(start string) (string, bool, error) {
	return findHarnessFile(start, "bridge.yaml")
}

func findHarnessFile(start, name string) (string, bool, error) {
	volume := filepath.VolumeName(start)
	for current := start; ; current = filepath.Dir(current) {
		if filepath.VolumeName(current) != volume {
			break
		}
		path := filepath.Join(current, ".harness", name)
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return "", false, fmt.Errorf("harness identity file must be a regular non-symlink file: %s", path)
			}
			return path, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", false, nil
}

func gitRead(ctx context.Context, directory string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func samePath(left, right string) bool {
	leftResolved, leftErr := filepath.EvalSymlinks(left)
	rightResolved, rightErr := filepath.EvalSymlinks(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftResolved) == filepath.Clean(rightResolved)
}

func pathInside(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
