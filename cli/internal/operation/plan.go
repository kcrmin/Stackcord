package operation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileChange is one exact file replacement relative to Plan.Root.
type FileChange struct {
	Path    string      `json:"path"`
	Content []byte      `json:"content"`
	Mode    os.FileMode `json:"mode"`
}

// Plan is a deterministic set of local mutations with a preflight fingerprint.
type Plan struct {
	ID                      string       `json:"id"`
	Root                    string       `json:"root"`
	InitialStateFingerprint string       `json:"initial_state_fingerprint"`
	Files                   []FileChange `json:"files"`
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
		ID    string `json:"id"`
		Files []file `json:"files"`
	}{plan.ID, files})
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
	target := filepath.Join(rootAbsolute, clean)
	rel, err := filepath.Rel(rootAbsolute, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target path %q escapes project root", relative)
	}
	return target, nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
