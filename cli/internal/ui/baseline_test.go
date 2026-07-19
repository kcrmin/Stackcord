package ui_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestUIBaselineIdentityIsSafeAndDeterministic(t *testing.T) {
	baseline := ui.Baseline{
		SchemaVersion:      1,
		ID:                 "ui.baseline.checkout",
		WorkspaceID:        "workspace.ui",
		WorkspaceCommit:    strings.Repeat("a", 40),
		WorkspaceRemote:    "https://example.test/product-ui.git",
		SourceIDs:          []string{"ui.external.checkout"},
		SourceFingerprints: map[string]string{"ui.external.checkout": "sha256:" + strings.Repeat("c", 64)},
		MappedRefs:         []string{"ui.checkout"},
		Consumers:          []string{"workspace.frontend"},
	}

	require.Empty(t, ui.ValidateBaseline(baseline))
	first := ui.BaselineFingerprint(baseline)
	reordered := baseline
	reordered.MappedRefs = []string{"ui.checkout"}
	require.Equal(t, first, ui.BaselineFingerprint(reordered))

	changed := baseline
	changed.WorkspaceCommit = strings.Repeat("b", 40)
	require.NotEqual(t, first, ui.BaselineFingerprint(changed))

	unsafe := baseline
	unsafe.WorkspaceRemote = "https://user:secret@example.test/product-ui.git"
	require.NotEmpty(t, ui.ValidateBaseline(unsafe))
}

func TestBindUIBaselineRequiresCleanPublishedWorkspaceCommit(t *testing.T) {
	root := t.TempDir()
	uiRoot := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.MkdirAll(uiRoot, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n  - id: workspace.ui\n    kind: directory\n    path: ui\n    remote: https://example.test/product-ui.git\n    responsibilities: [ui-baseline]\n    dependencies: [workspace.root]\n"), 0o600))
	uiGit(t, uiRoot, "init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(uiRoot, "README.md"), []byte("UI\n"), 0o600))
	uiGit(t, uiRoot, "add", ".")
	uiGit(t, uiRoot, "commit", "-m", "docs: define UI baseline")
	uiGit(t, uiRoot, "remote", "add", "origin", "https://example.test/product-ui.git")
	uiGit(t, uiRoot, "update-ref", "refs/remotes/origin/main", "HEAD")

	baseline, plan, warnings, err := ui.PlanBaseline(context.Background(), ui.BaselineRequest{Root: root, ID: "ui.baseline.checkout", WorkspaceID: "workspace.ui", MappedRefs: []string{"ui.checkout"}, Consumers: []string{"workspace.frontend"}})
	require.NoError(t, err)
	require.Empty(t, plan.Blockers)
	require.Empty(t, warnings)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	loaded, err := ui.LoadBaseline(root, baseline.ID)
	require.NoError(t, err)
	require.Equal(t, baseline.WorkspaceCommit, loaded.WorkspaceCommit)

	require.NoError(t, os.WriteFile(filepath.Join(uiRoot, "dirty.txt"), []byte("dirty"), 0o600))
	_, dirtyPlan, _, err := ui.PlanBaseline(context.Background(), ui.BaselineRequest{Root: root, ID: "ui.baseline.checkout", WorkspaceID: "workspace.ui", MappedRefs: []string{"ui.checkout"}, Consumers: []string{"workspace.frontend"}})
	require.NoError(t, err)
	require.NotEmpty(t, dirtyPlan.Blockers)
}

func uiGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.test", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.test")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
