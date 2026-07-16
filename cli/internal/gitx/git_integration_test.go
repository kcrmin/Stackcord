package gitx_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/gitx"
	"github.com/stretchr/testify/require"
)

func TestInspectIsReadOnlyAndReportsDivergence(t *testing.T) {
	root, remote := repositoryFixture(t)
	other := filepath.Join(t.TempDir(), "other")
	runGit(t, "", "clone", remote, other)
	configureGit(t, other)
	require.NoError(t, os.WriteFile(filepath.Join(other, "remote.txt"), []byte("remote\n"), 0o600))
	runGit(t, other, "add", "remote.txt")
	runGit(t, other, "commit", "-m", "feat: add remote file")
	runGit(t, other, "push")
	runGit(t, root, "fetch", "origin")

	require.NoError(t, os.WriteFile(filepath.Join(root, "local.txt"), []byte("local\n"), 0o600))
	runGit(t, root, "add", "local.txt")
	runGit(t, root, "commit", "-m", "feat: add local file")
	require.NoError(t, os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty\n"), 0o600))

	beforeReflog := runGit(t, root, "reflog", "--format=%H")
	beforeStatus := runGit(t, root, "status", "--porcelain=v2", "--untracked-files=all")
	beforeIndex := fileHash(t, filepath.Join(root, ".git", "index"))

	state, err := gitx.Inspect(context.Background(), root)
	require.NoError(t, err)
	require.Equal(t, "main", state.Branch)
	require.True(t, state.Dirty)
	require.Equal(t, 1, state.Ahead)
	require.Equal(t, 1, state.Behind)
	require.True(t, state.Diverged)

	require.Equal(t, beforeReflog, runGit(t, root, "reflog", "--format=%H"))
	require.Equal(t, beforeStatus, runGit(t, root, "status", "--porcelain=v2", "--untracked-files=all"))
	require.Equal(t, beforeIndex, fileHash(t, filepath.Join(root, ".git", "index")))
}

func TestInspectAndPlanMissingSubmoduleAtPinnedCommit(t *testing.T) {
	child, _ := repositoryFixture(t)
	root, _ := repositoryFixture(t)
	runGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", child, "services/identity")
	runGit(t, root, "config", "-f", ".gitmodules", "submodule.services/identity.url", "https://example.invalid/identity.git")
	runGit(t, root, "commit", "-am", "build: add identity workspace")
	require.NoError(t, os.RemoveAll(filepath.Join(root, "services", "identity")))

	state, err := gitx.Inspect(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, state.Submodules, 1)
	require.False(t, state.Submodules[0].Initialized)
	require.NotEmpty(t, state.Submodules[0].ExpectedSHA)

	plan := gitx.PlanWorkspaceSync(state)
	require.Len(t, plan.Commands, 1)
	require.Equal(t, []string{"submodule", "update", "--init", "--recursive", "--", "services/identity"}, plan.Commands[0].Args)
}

func TestPlanWorktreeUsesConventionalBranchAndOutsidePath(t *testing.T) {
	root, _ := repositoryFixture(t)
	plan, err := gitx.PlanWorktree(gitx.WorktreeChange{Root: root, Branch: "feature/GH-142-account-recovery"})
	require.NoError(t, err)
	require.Len(t, plan.Commands, 1)
	require.NotContains(t, plan.Commands[0].Args[2], root+string(filepath.Separator))

	_, err = gitx.PlanWorktree(gitx.WorktreeChange{Root: root, Branch: "agent/generated-work"})
	require.ErrorContains(t, err, "branch must match")
}

func TestPlanWorkspaceSyncBlocksUnsafeOrLocallyChangedSubmodules(t *testing.T) {
	state := gitx.State{Root: t.TempDir(), Submodules: []gitx.Submodule{
		{Path: "services/unsafe", URL: "file:///tmp/unsafe", UnsafeURL: true},
		{Path: "services/dirty", URL: "https://example.invalid/dirty.git", Initialized: true, Dirty: true},
		{Path: "services/mismatch", URL: "https://example.invalid/mismatch.git", Initialized: true, PointerDiff: true},
	}}
	plan := gitx.PlanWorkspaceSync(state)
	require.Empty(t, plan.Commands)
	require.Len(t, plan.Blockers, 3)
	require.Equal(t, "git.submodule.unsafe-url", plan.Blockers[0].Code)
}

func repositoryFixture(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	root := filepath.Join(base, "work")
	runGit(t, "", "init", "--bare", "--initial-branch=main", remote)
	runGit(t, "", "init", "--initial-branch=main", root)
	configureGit(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600))
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "chore: initialize fixture")
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "-u", "origin", "main")
	return root, remote
}

func configureGit(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "config", "user.email", "fixture@example.invalid")
	runGit(t, root, "config", "user.name", "Fixture User")
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	if root != "" {
		command.Dir = root
	}
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
