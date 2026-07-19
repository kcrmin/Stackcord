package gitx_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
	"github.com/stretchr/testify/require"
)

func TestCreateWorktreeRefusesDirtyBaseAndDuplicateBranch(t *testing.T) {
	root, _ := repositoryFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty\n"), 0o600))

	dirty := gitx.CreateWorktree(context.Background(), gitx.CreateWorktreeRequest{Root: root, Branch: "feature/account-recovery", Base: "main"})

	require.Equal(t, domain.StatusBlocked, dirty.Status)
	require.Contains(t, gitResultCodes(dirty.Blockers), "git.base-dirty")
	require.NoDirExists(t, filepath.Join(filepath.Dir(root), ".stackcord-worktrees", filepath.Base(root), "feature-account-recovery"))
	require.NoError(t, os.Remove(filepath.Join(root, "dirty.txt")))

	created := gitx.CreateWorktree(context.Background(), gitx.CreateWorktreeRequest{Root: root, Branch: "feature/account-recovery", Base: "main"})
	require.Equal(t, domain.StatusPassed, created.Status)
	duplicate := gitx.CreateWorktree(context.Background(), gitx.CreateWorktreeRequest{Root: root, Branch: "feature/account-recovery", Base: "main"})
	require.Equal(t, domain.StatusBlocked, duplicate.Status)
	require.Contains(t, gitResultCodes(duplicate.Blockers), "git.branch-in-use")
}

func TestCreateWorktreeRejectsTargetInsideAnotherRepository(t *testing.T) {
	root, _ := repositoryFixture(t)
	other, _ := repositoryFixture(t)
	target := filepath.Join(other, "nested", "account-recovery")

	result := gitx.CreateWorktree(context.Background(), gitx.CreateWorktreeRequest{Root: root, Branch: "feature/account-recovery", Base: "main", Target: target})

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, gitResultCodes(result.Blockers), "git.worktree-target-unsafe")
	require.NoDirExists(t, target)
}

func TestSyncPinnedSubmoduleVerifiesExactPostcondition(t *testing.T) {
	root, expected := pinnedSubmoduleFixture(t, true)

	result := gitx.SyncPinnedSubmodules(context.Background(), root, []string{"backend"})

	require.Equal(t, domain.StatusPassed, result.Status, result.Blockers)
	require.Equal(t, expected, runGit(t, filepath.Join(root, "backend"), "rev-parse", "HEAD"))
	require.Contains(t, gitResultCodes(result.Evidence), "git.submodule-pinned")
}

func TestSyncPinnedSubmoduleRejectsGitmoduleMissingHarnessDeclaration(t *testing.T) {
	root, _ := pinnedSubmoduleFixture(t, false)

	result := gitx.SyncPinnedSubmodules(context.Background(), root, []string{"backend"})

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, gitResultCodes(result.Blockers), "git.submodule-not-in-workspace-manifest")
	require.NoDirExists(t, filepath.Join(root, "backend"))
}

func TestSyncPinnedSubmoduleRejectsMixedSafeAndEscapingPaths(t *testing.T) {
	root, _ := pinnedSubmoduleFixture(t, true)

	result := gitx.SyncPinnedSubmodules(context.Background(), root, []string{"backend", "../outside"})

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, gitResultCodes(result.Blockers), "git.submodule-path-invalid")
	require.NoDirExists(t, filepath.Join(root, "backend"))
}

func pinnedSubmoduleFixture(t *testing.T, declared bool) (string, string) {
	t.Helper()
	child, _ := repositoryFixture(t)
	root, _ := repositoryFixture(t)
	runGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", child, "backend")
	runGit(t, root, "config", "-f", ".gitmodules", "submodule.backend.url", "https://example.invalid/backend.git")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	workspaces := "schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"
	if declared {
		workspaces += "  - id: workspace.backend\n    kind: submodule\n    path: backend\n    responsibilities: [backend]\n    dependencies: []\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(workspaces), 0o600))
	runGit(t, root, "add", ".harness/workspaces.yaml")
	runGit(t, root, "commit", "-am", "build: add backend workspace")
	expected := runGit(t, root, "rev-parse", "HEAD:backend")
	runGit(t, root, "submodule", "deinit", "-f", "--", "backend")
	runGit(t, root, "config", "submodule.backend.url", "https://example.invalid/backend.git")
	require.NoError(t, os.RemoveAll(filepath.Join(root, "backend")))
	return root, expected
}

func gitResultCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
