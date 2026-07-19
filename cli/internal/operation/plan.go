package operation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

var operationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// FileChange is one exact file replacement relative to Plan.Root.
type FileChange struct {
	Path    string      `json:"path"`
	Content []byte      `json:"content"`
	Mode    os.FileMode `json:"mode"`
}

// CommandStep is an explicit, reviewable external command. Apply never executes it implicitly.
type CommandStep struct {
	Program       string   `json:"program"`
	Args          []string `json:"args"`
	Directory     string   `json:"directory"`
	ApprovalClass string   `json:"approval_class"`
}

// Plan is a deterministic set of local mutations with a preflight fingerprint.
type Plan struct {
	ID                      string        `json:"id"`
	Root                    string        `json:"root"`
	InitialStateFingerprint string        `json:"initial_state_fingerprint"`
	Files                   []FileChange  `json:"files"`
	Commands                []CommandStep `json:"commands,omitempty"`
	Blockers                []domain.Item `json:"blockers,omitempty"`
}

// StateFingerprint captures the target paths before mutation.
func StateFingerprint(plan Plan) (string, error) {
	type state struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
		Hash string `json:"hash,omitempty"`
	}
	states := make([]state, 0, len(plan.Files))
	for _, change := range plan.Files {
		path, err := safeTarget(plan.Root, change.Path)
		if err != nil {
			return "", err
		}
		current := state{Path: filepath.ToSlash(change.Path), Kind: "missing"}
		info, statErr := os.Lstat(path)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("target %s is a symlink", change.Path)
			}
			if info.IsDir() {
				current.Kind = "directory"
			} else {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return "", readErr
				}
				current.Kind = "file"
				current.Hash = digest(data)
			}
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		states = append(states, current)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Path < states[j].Path })
	data, err := json.Marshal(states)
	if err != nil {
		return "", err
	}
	return digest(data), nil
}

func validatePlanIdentity(plan Plan) error {
	if plan.Root == "" {
		return fmt.Errorf("operation root is required")
	}
	if !operationIDPattern.MatchString(plan.ID) {
		return fmt.Errorf("operation ID must contain only 1-128 letters, digits, dots, underscores, or hyphens")
	}
	return nil
}

func planFingerprint(plan Plan) string {
	type file struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
		Mode uint32 `json:"mode"`
	}
	files := make([]file, 0, len(plan.Files))
	for _, change := range plan.Files {
		files = append(files, file{Path: filepath.ToSlash(change.Path), Hash: digest(change.Content), Mode: uint32(change.Mode.Perm())})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	data, _ := json.Marshal(struct {
		ID       string        `json:"id"`
		Files    []file        `json:"files"`
		Commands []CommandStep `json:"commands,omitempty"`
		Blockers []domain.Item `json:"blockers,omitempty"`
	}{plan.ID, files, plan.Commands, plan.Blockers})
	return digest(data)
}

func safeTarget(root, relative string) (string, error) {
	if root == "" || relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("target path %q escapes project root", relative)
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target path %q escapes project root", relative)
	}
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbsolute, err = resolveExistingAncestor(rootAbsolute)
	if err != nil {
		return "", err
	}
	target := filepath.Join(rootAbsolute, clean)
	rel, err := filepath.Rel(rootAbsolute, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target path %q escapes project root", relative)
	}
	current := rootAbsolute
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("target path %q crosses symlink %s", relative, current)
		}
	}
	return target, nil
}

func resolveExistingAncestor(value string) (string, error) {
	current := filepath.Clean(value)
	missing := []string{}
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return resolved, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing ancestor for %s", value)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
