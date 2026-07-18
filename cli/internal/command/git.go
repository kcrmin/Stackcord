package command

import (
	"strconv"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"github.com/spf13/cobra"
)

func newGitCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "git", Short: "Inspect Git state and apply narrowly allow-listed workspace operations"}
	var root string
	inspect := &cobra.Command{Use: "inspect", RunE: func(cmd *cobra.Command, _ []string) error {
		state, err := gitx.Inspect(cmd.Context(), root)
		if err != nil {
			return err
		}
		status := domain.StatusPassed
		summary := "Git state is clean and coordinated."
		if state.Dirty || state.Diverged || state.Detached {
			status, summary = domain.StatusWarning, "Git state needs review before shared mutation."
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "git.inspect", OperationID: "git-inspect-read-only", Status: status, ExitCode: domain.ExitSuccess, Summary: summary, Project: &domain.Project{Root: state.Root}, Facts: []domain.Item{
			{Code: "git.branch", Message: state.Branch}, {Code: "git.head", Message: state.Head}, {Code: "git.upstream", Message: state.Upstream},
			{Code: "git.dirty", Message: strconv.FormatBool(state.Dirty)}, {Code: "git.detached", Message: strconv.FormatBool(state.Detached)},
			{Code: "git.ahead", Message: strconv.Itoa(state.Ahead)}, {Code: "git.behind", Message: strconv.Itoa(state.Behind)}, {Code: "git.diverged", Message: strconv.FormatBool(state.Diverged)},
			{Code: "git.worktrees", Message: strconv.Itoa(len(state.Worktrees))}, {Code: "git.submodules", Message: strconv.Itoa(len(state.Submodules))},
		}}
		for _, worktree := range state.Worktrees {
			result.Facts = append(result.Facts,
				domain.Item{Code: "git.worktree", Message: worktree.Path, Refs: []string{worktree.Branch, worktree.Head}},
				domain.Item{Code: "git.worktree.detached", Message: strconv.FormatBool(worktree.Detached), Refs: []string{worktree.Path}},
				domain.Item{Code: "git.worktree.locked", Message: strconv.FormatBool(worktree.Locked), Refs: []string{worktree.Path}},
			)
		}
		for _, submodule := range state.Submodules {
			refs := []string{submodule.Path}
			result.Facts = append(result.Facts,
				domain.Item{Code: "git.submodule", Message: submodule.Path, Refs: []string{submodule.ExpectedSHA, submodule.Head}},
				domain.Item{Code: "git.submodule.expected-head", Message: submodule.ExpectedSHA, Refs: refs},
				domain.Item{Code: "git.submodule.actual-head", Message: submodule.Head, Refs: refs},
				domain.Item{Code: "git.submodule.initialized", Message: strconv.FormatBool(submodule.Initialized), Refs: refs},
				domain.Item{Code: "git.submodule.dirty", Message: strconv.FormatBool(submodule.Dirty), Refs: refs},
				domain.Item{Code: "git.submodule.pointer-mismatch", Message: strconv.FormatBool(submodule.PointerDiff), Refs: refs},
			)
			if !submodule.Initialized {
				result.Warnings = append(result.Warnings, domain.Item{Code: "git.submodule-missing", Message: "Submodule is not initialized at the root-recorded commit.", Refs: refs})
			}
			if submodule.Dirty {
				result.Warnings = append(result.Warnings, domain.Item{Code: "git.submodule-dirty", Message: "Submodule has local changes.", Refs: refs})
			}
			if submodule.PointerDiff {
				result.Warnings = append(result.Warnings, domain.Item{Code: "git.submodule-pointer-mismatch", Message: "Submodule HEAD differs from the root-recorded commit.", Refs: []string{submodule.Path, submodule.ExpectedSHA, submodule.Head}})
			}
			if submodule.UnsafeURL {
				result.Warnings = append(result.Warnings, domain.Item{Code: "git.submodule-unsafe-url", Message: "Submodule URL requires explicit security review.", Refs: refs})
			}
		}
		if len(result.Warnings) > 0 && result.Status == domain.StatusPassed {
			result.Status, result.Summary = domain.StatusWarning, "Git or submodule state needs review before shared mutation."
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	inspect.Flags().StringVar(&root, "root", ".", "repository path")
	parent.AddCommand(inspect)
	parent.AddCommand(newGitSyncPlan(version, jsonOutput))
	parent.AddCommand(newGitSync(version, jsonOutput))
	parent.AddCommand(newGitWorktreePlan(version, jsonOutput))
	parent.AddCommand(newGitWorktree(version, jsonOutput))
	return parent
}

func newGitSync(version string, jsonOutput *bool) *cobra.Command {
	var root string
	var paths []string
	var apply bool
	command := &cobra.Command{Use: "sync", Short: "Initialize explicit submodules at exact root pointers", RunE: func(cmd *cobra.Command, _ []string) error {
		state, err := gitx.Inspect(cmd.Context(), root)
		if err != nil {
			return err
		}
		if !apply {
			return writeResult(cmd, *jsonOutput, planResult(version, "git.sync-plan", gitx.PlanWorkspaceSync(state), "Pinned workspace synchronization plan is ready; no Git mutation was attempted."))
		}
		result := gitx.SyncPinnedSubmodules(cmd.Context(), root, paths)
		result.ToolVersion, result.Command = version, "git.sync"
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "exact orchestration repository root")
	command.Flags().StringSliceVar(&paths, "path", nil, "explicit declared submodule path")
	command.Flags().BoolVar(&apply, "apply", false, "execute the reviewed allow-listed pinned sync")
	return command
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

func newGitWorktree(version string, jsonOutput *bool) *cobra.Command {
	var root, branch, base, target string
	var apply bool
	command := &cobra.Command{Use: "worktree", Short: "Create and verify an isolated conventional feature worktree", RunE: func(cmd *cobra.Command, _ []string) error {
		if !apply {
			plan, err := gitx.PlanWorktree(gitx.WorktreeChange{Root: root, Branch: branch, Base: base, Target: target})
			if err != nil {
				return err
			}
			return writeResult(cmd, *jsonOutput, planResult(version, "git.worktree-plan", plan, "Isolated worktree plan is ready; no Git mutation was attempted."))
		}
		result := gitx.CreateWorktree(cmd.Context(), gitx.CreateWorktreeRequest{Root: root, Branch: branch, Base: base, Target: target})
		result.ToolVersion, result.Command = version, "git.worktree"
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "exact repository root")
	command.Flags().StringVar(&branch, "branch", "", "conventional branch name")
	command.Flags().StringVar(&base, "base", "main", "reviewed base branch or commit")
	command.Flags().StringVar(&target, "target", "", "optional explicit worktree target outside every repository")
	command.Flags().BoolVar(&apply, "apply", false, "execute the reviewed allow-listed worktree creation")
	_ = command.MarkFlagRequired("branch")
	return command
}
