package gitx

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/harness/convention"
	"github.com/kcrmin/Stackcord/cli/internal/harness/operation"
)

// WorktreeChange describes isolated work to be planned.
type WorktreeChange struct {
	Root   string
	Branch string
	Base   string
	Target string
}

// PlanWorktree validates conventions and places the worktree outside the repository.
func PlanWorktree(change WorktreeChange) (operation.Plan, error) {
	root, err := filepath.Abs(change.Root)
	if err != nil {
		return operation.Plan{}, err
	}
	if err := convention.ValidateBranch(root, change.Branch); err != nil {
		return operation.Plan{}, fmt.Errorf("branch must match the configured Git convention: %w", err)
	}
	if change.Base == "" {
		change.Base = "main"
	}
	if !safeBaseRef(change.Base) {
		return operation.Plan{}, fmt.Errorf("worktree base ref is invalid")
	}
	branchKey := strings.ReplaceAll(change.Branch, "/", "-")
	target, err := worktreeTarget(context.Background(), runner{}, root, change.Branch, change.Target)
	if err != nil {
		return operation.Plan{}, err
	}
	return operation.Plan{ID: "worktree-" + branchKey, Root: root, Commands: []operation.CommandStep{{Program: "git", Args: []string{"worktree", "add", "-b", change.Branch, target, change.Base}, Directory: root, ApprovalClass: "B"}}}, nil
}
