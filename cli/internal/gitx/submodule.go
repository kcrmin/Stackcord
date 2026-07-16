package gitx

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
)

// Submodule captures root pointer, local checkout, URL, and safety state.
type Submodule struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	ExpectedSHA string `json:"expected_sha"`
	Head        string `json:"head,omitempty"`
	Initialized bool   `json:"initialized"`
	Dirty       bool   `json:"dirty"`
	PointerDiff bool   `json:"pointer_diff"`
	UnsafeURL   bool   `json:"unsafe_url"`
}

func inspectSubmodules(ctx context.Context, git runner, root string) ([]Submodule, error) {
	configured, err := parseGitmodules(filepath.Join(root, ".gitmodules"))
	if os.IsNotExist(err) {
		return []Submodule{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Submodule, 0, len(configured))
	for _, submodule := range configured {
		entry, treeErr := git.read(ctx, root, "ls-tree", "HEAD", "--", submodule.Path)
		if treeErr != nil {
			return nil, treeErr
		}
		fields := strings.Fields(entry)
		if len(fields) >= 3 && fields[1] == "commit" {
			submodule.ExpectedSHA = fields[2]
		}
		checkout := filepath.Join(root, filepath.FromSlash(submodule.Path))
		if _, statErr := os.Stat(checkout); statErr == nil {
			if head, headErr := git.read(ctx, checkout, "rev-parse", "HEAD"); headErr == nil {
				submodule.Initialized = true
				submodule.Head = head
				submodule.PointerDiff = head != submodule.ExpectedSHA
				status, statusErr := git.read(ctx, checkout, "status", "--porcelain=v2", "--untracked-files=all")
				submodule.Dirty = statusErr != nil || status != ""
			}
		}
		submodule.UnsafeURL = unsafeSubmoduleURL(submodule.URL)
		result = append(result, submodule)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func parseGitmodules(path string) ([]Submodule, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var result []Submodule
	var current *Submodule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[submodule \"") && strings.HasSuffix(line, "\"]") {
			name := strings.TrimSuffix(strings.TrimPrefix(line, "[submodule \""), "\"]")
			result = append(result, Submodule{Name: name})
			current = &result[len(result)-1]
			continue
		}
		if current == nil {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "path":
			current.Path = filepath.ToSlash(strings.TrimSpace(value))
		case "url":
			current.URL = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, submodule := range result {
		if submodule.Path == "" || filepath.IsAbs(submodule.Path) || strings.HasPrefix(filepath.Clean(submodule.Path), "..") {
			return nil, fmt.Errorf("unsafe submodule path %q", submodule.Path)
		}
	}
	return result, nil
}

func unsafeSubmoduleURL(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	return lower == "" || strings.HasPrefix(lower, "-") || strings.HasPrefix(lower, "ext::") || strings.HasPrefix(lower, "file:") || filepath.IsAbs(url)
}

// PlanWorkspaceSync returns exact pinned update commands without executing them.
func PlanWorkspaceSync(state State) operation.Plan {
	plan := operation.Plan{ID: "workspace-sync", Root: state.Root}
	for _, submodule := range state.Submodules {
		if submodule.UnsafeURL {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule.unsafe-url", Message: "Submodule URL requires explicit security review.", Refs: []string{submodule.Path}})
			continue
		}
		if submodule.Dirty {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule.dirty", Message: "Submodule has local changes and cannot be synchronized automatically.", Refs: []string{submodule.Path}})
			continue
		}
		if submodule.Initialized && submodule.PointerDiff {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule.pointer-mismatch", Message: "Submodule HEAD differs from the root gitlink pointer.", Refs: []string{submodule.Path, submodule.ExpectedSHA, submodule.Head}})
			continue
		}
		if !submodule.Initialized {
			plan.Commands = append(plan.Commands, operation.CommandStep{Program: "git", Args: []string{"submodule", "update", "--init", "--", submodule.Path}, Directory: state.Root, ApprovalClass: "C"})
		}
	}
	return plan
}
