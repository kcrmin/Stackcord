package gitx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubmoduleAddPlanAllowsOnlySafeReviewedBoundary(t *testing.T) {
	root := submoduleAddRepository(t)
	request := SubmoduleAddRequest{Root: root, Remote: "https://example.test/product-ui.git", Path: "ui"}

	plan := PlanSubmoduleAdd(context.Background(), request)
	require.Empty(t, plan.Blockers)
	require.Len(t, plan.Commands, 1)
	require.Equal(t, []string{"submodule", "add", "--", request.Remote, request.Path}, plan.Commands[0].Args)

	for _, remote := range []string{"file:///tmp/ui.git", "../ui", "https://user:secret@example.test/ui.git", "-c"} {
		request.Remote = remote
		blocked := PlanSubmoduleAdd(context.Background(), request)
		require.NotEmpty(t, blocked.Blockers, remote)
	}

	require.NoDirExists(t, filepath.Join(root, "ui"), "planning must not mutate Git")
}

func submoduleAddRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.test", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.test")
		require.NoError(t, command.Run())
	}
	run("init", "-b", "main")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nroot_remote: https://example.test/root.git\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	run("add", ".")
	run("commit", "-m", "chore: initialize project")
	return root
}
