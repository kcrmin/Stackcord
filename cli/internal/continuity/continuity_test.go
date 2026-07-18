package continuity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestCollectDistinguishesUnknownAndLocalOnlyEvidence(t *testing.T) {
	root := continuityFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "local-only.txt"), []byte("local\n"), 0o600))

	got := Collect(context.Background(), root, Options{})

	require.Equal(t, "project.example", got.ProjectID)
	require.Equal(t, Unknown, got.Overall)
	require.Contains(t, issueCodes(got.Issues), "provider.live-unknown")
	require.Contains(t, issueCodes(got.Issues), "workspace.local-only")
	require.Contains(t, issueCodes(got.Issues), "workspace.dirty")
	require.Len(t, got.NextActions, 1)
}

func TestWorkspaceCollectionBlocksPointerMismatch(t *testing.T) {
	root := t.TempDir()
	manifest := workspace.Manifest{ProjectID: "project.example", Workspaces: []workspace.Entry{
		{ID: "workspace.root", Kind: "root", Path: "."},
		{ID: "workspace.backend", Kind: "submodule", Path: "backend"},
	}}
	rootGit := gitx.State{Root: root, Head: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Submodules: []gitx.Submodule{{
		Path: "backend", Initialized: true, PointerDiff: true,
		ExpectedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Head:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}}

	_, issues := collectWorkspaceStates(context.Background(), root, manifest, rootGit)

	require.Contains(t, issueCodes(issues), "workspace.pointer-mismatch")
}

func continuityFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{
		filepath.Join(root, ".harness", "work"),
		filepath.Join(root, "specs", "policies"),
		filepath.Join(root, "contracts"),
	} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\n"), 0o600))
	policy := "---\nschema_version: 1\nid: policy.example\nkind: policy\nstatus: approved\nrevision: 1\nrefs: []\n---\nExample.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "example.md"), []byte(policy), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte("schema_version: 1\ncontracts: []\n"), 0o600))
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "chore: initialize project")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func issueCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
