package gitx

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/operation"
)

var branchPattern = regexp.MustCompile(`^(feature|fix|bugfix|chore|docs|refactor|test|release)/([A-Za-z0-9]+-)?[a-z0-9]+(?:-[a-z0-9]+)*$`)

// WorktreeChange describes isolated work to be planned.
type WorktreeChange struct {
	Root   string
	Branch string
	Base   string
	Target string
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

// ValidateBranch enforces the repository-neutral collaboration convention.
func ValidateBranch(branch string) error {
	if !branchPattern.MatchString(branch) || containsAIMarker(branch) {
		return fmt.Errorf("invalid conventional branch")
	}
	return nil
}

func containsAIMarker(branch string) bool {
	lower := strings.ToLower(branch)
	description := lower
	if _, value, found := strings.Cut(lower, "/"); found {
		description = value
	}
	for _, token := range strings.Split(description, "-") {
		if token == "ai" || token == "agent" || token == "codex" || token == "gpt" {
			return true
		}
	}
	return strings.Contains(description, "generated-by") || strings.Contains(description, "model-generated") || strings.Contains(description, "generated-model")
}
