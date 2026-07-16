package gitx

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"fullstack-orchestrator/cli/internal/operation"
)

var branchPattern = regexp.MustCompile(`^(feature|fix|bugfix|chore|docs|refactor|test|release)/([A-Za-z0-9]+-)?[a-z0-9]+(?:-[a-z0-9]+)*$`)

// WorktreeChange describes isolated work to be planned.
type WorktreeChange struct {
	Root   string
	Branch string
}

// PlanWorktree validates conventions and places the worktree outside the repository.
func PlanWorktree(change WorktreeChange) (operation.Plan, error) {
	if err := ValidateBranch(change.Branch); err != nil {
		return operation.Plan{}, fmt.Errorf("branch must match <type>/<description> or <type>/<work-id>-<description> without AI markers")
	}
	root, err := filepath.Abs(change.Root)
	if err != nil {
		return operation.Plan{}, err
	}
	repositoryName := filepath.Base(root)
	branchKey := strings.ReplaceAll(change.Branch, "/", "-")
	target := filepath.Join(filepath.Dir(root), ".orchestrator-worktrees", repositoryName, branchKey)
	return operation.Plan{ID: "worktree-" + branchKey, Root: root, Commands: []operation.CommandStep{{Program: "git", Args: []string{"worktree", "add", target, "-b", change.Branch}, Directory: root, ApprovalClass: "B"}}}, nil
}

// ValidateBranch enforces the repository-neutral collaboration convention.
func ValidateBranch(branch string) error {
	if !branchPattern.MatchString(branch) || containsAIMarker(branch) {
		return fmt.Errorf("invalid conventional branch")
	}
	return nil
}

func containsAIMarker(branch string) bool {
	lower := strings.ToLower(branch)
	for _, marker := range []string{"ai-", "agent-", "codex-", "-ai", "-agent", "-codex"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
