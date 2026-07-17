package gitx

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Worktree is one actual linked checkout reported by Git.
type Worktree struct {
	Path     string `json:"path"`
	Head     string `json:"head"`
	Branch   string `json:"branch,omitempty"`
	Detached bool   `json:"detached"`
	Locked   bool   `json:"locked"`
}

func inspectWorktrees(ctx context.Context, git runner, root string) ([]Worktree, error) {
	output, err := git.read(ctx, root, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	result := []Worktree{}
	var current *Worktree
	for _, field := range strings.Split(output, "\x00") {
		if field == "" {
			if current != nil {
				result = append(result, *current)
				current = nil
			}
			continue
		}
		key, value, _ := strings.Cut(field, " ")
		if key == "worktree" {
			if current != nil {
				return nil, fmt.Errorf("invalid Git worktree record")
			}
			current = &Worktree{Path: value}
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("invalid Git worktree record")
		}
		switch key {
		case "HEAD":
			current.Head = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "detached":
			current.Detached = true
		case "locked":
			current.Locked = true
		}
	}
	if current != nil {
		result = append(result, *current)
	}
	for _, worktree := range result {
		if worktree.Path == "" || worktree.Head == "" {
			return nil, fmt.Errorf("git worktree record is incomplete")
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}
