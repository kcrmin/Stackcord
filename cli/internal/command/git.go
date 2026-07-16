package command

import (
	"strconv"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"github.com/spf13/cobra"
)

func newGitCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "git", Short: "Inspect Git, submodule, and worktree actual state without mutation"}
	var root string
	inspect := &cobra.Command{Use: "inspect", RunE: func(cmd *cobra.Command, _ []string) error {
		state, err := gitx.Inspect(cmd.Context(), root)
		if err != nil {
			return err
		}
		status := domain.StatusPassed
		summary := "Git state is clean and coordinated."
		if state.Dirty || state.Diverged {
			status, summary = domain.StatusWarning, "Git state needs review before shared mutation."
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "git.inspect", OperationID: "git-inspect-read-only", Status: status, ExitCode: domain.ExitSuccess, Summary: summary, Project: &domain.Project{Root: state.Root}, Facts: []domain.Item{
			{Code: "git.branch", Message: state.Branch}, {Code: "git.head", Message: state.Head}, {Code: "git.dirty", Message: strconv.FormatBool(state.Dirty)}, {Code: "git.ahead", Message: strconv.Itoa(state.Ahead)}, {Code: "git.behind", Message: strconv.Itoa(state.Behind)}, {Code: "git.submodules", Message: strconv.Itoa(len(state.Submodules))},
		}}
		for _, submodule := range state.Submodules {
			result.Facts = append(result.Facts, domain.Item{Code: "git.submodule", Message: submodule.Path, Refs: []string{submodule.ExpectedSHA, submodule.Head}})
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	inspect.Flags().StringVar(&root, "root", ".", "repository path")
	parent.AddCommand(inspect)
	parent.AddCommand(newGitSyncPlan(version, jsonOutput))
	parent.AddCommand(newGitWorktreePlan(version, jsonOutput))
	return parent
}

func newGitSyncPlan(version string, jsonOutput *bool) *cobra.Command {
	var root string
	command := &cobra.Command{Use: "sync-plan", Short: "Plan exact root-pinned submodule initialization without mutation", RunE: func(cmd *cobra.Command, _ []string) error {
		state, err := gitx.Inspect(cmd.Context(), root)
		if err != nil {
			return err
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "git.sync-plan", gitx.PlanWorkspaceSync(state), "Pinned workspace synchronization plan is ready."))
	}}
	command.Flags().StringVar(&root, "root", ".", "repository path")
	return command
}

func newGitWorktreePlan(version string, jsonOutput *bool) *cobra.Command {
	var root, branch string
	command := &cobra.Command{Use: "worktree-plan", Short: "Plan an isolated conventional feature worktree", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := gitx.PlanWorktree(gitx.WorktreeChange{Root: root, Branch: branch})
		if err != nil {
			return err
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "git.worktree-plan", plan, "Isolated worktree plan is ready."))
	}}
	command.Flags().StringVar(&root, "root", ".", "repository path")
	command.Flags().StringVar(&branch, "branch", "", "conventional branch name")
	_ = command.MarkFlagRequired("branch")
	return command
}
