package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/stretchr/testify/require"
)

func TestRegisterUIWorkspacePreservesProjectAndLinksFrontend(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(`schema_version: 1
project_id: project.example
root_remote: https://example.test/root.git
workspaces:
  - id: workspace.root
    kind: root
    path: .
    responsibilities: [orchestration]
    dependencies: []
  - id: workspace.frontend
    kind: directory
    path: frontend
    responsibilities: [frontend]
    dependencies: [workspace.root]
`), 0o600))

	plan, err := PlanRegistration(context.Background(), RegistrationRequest{
		Root: root, ID: "workspace.ui", Kind: "directory", Path: "ui",
		Responsibilities: []string{"ui-baseline"}, Dependencies: []string{"workspace.root"},
		Consumers: []string{"workspace.frontend"}, Initialize: "ui",
	})
	require.NoError(t, err)
	require.Empty(t, plan.Blockers)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	manifest, err := Load(root)
	require.NoError(t, err)
	require.Equal(t, []string{"workspace.root", "workspace.ui"}, manifestEntry(t, manifest, "workspace.frontend").Dependencies)
	require.Equal(t, []string{"ui-baseline"}, manifestEntry(t, manifest, "workspace.ui").Responsibilities)
	require.FileExists(t, filepath.Join(root, "ui", "README.md"))
	require.FileExists(t, filepath.Join(root, "ui", "coverage", "index.yaml"))

	again, err := PlanRegistration(context.Background(), RegistrationRequest{Root: root, ID: "workspace.other", Kind: "directory", Path: "ui", Responsibilities: []string{"ui-baseline"}})
	require.NoError(t, err)
	require.NotEmpty(t, again.Blockers, "duplicate paths must not replace existing workspaces")
}

func manifestEntry(t *testing.T, manifest Manifest, id string) Entry {
	t.Helper()
	for _, entry := range manifest.Workspaces {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("workspace %s not found", id)
	return Entry{}
}
