package workspace

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindRootFromSubmoduleUsesActualSuperproject(t *testing.T) {
	root, child := newRootWithChildSubmodule(t)

	got, err := FindRoot(context.Background(), child)

	require.NoError(t, err)
	require.Equal(t, root, got.Path)
	require.Equal(t, "workspace.backend", got.CurrentWorkspaceID)
	require.Equal(t, RootFromSuperproject, got.Source)
}

func TestFindRootFromStandaloneChildReportsIncompleteContext(t *testing.T) {
	child := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(child, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(child, ".harness", "bridge.yaml"), []byte(`schema_version: 1
project_id: project.example
root_remote: https://example.invalid/root.git
workspace_id: workspace.backend
discovery: git-superproject
contract_fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
commands_path: .harness/commands.yaml
`), 0o600))

	_, err := FindRoot(context.Background(), child)

	var incomplete *IncompleteContextError
	require.ErrorAs(t, err, &incomplete)
	require.Equal(t, "project.example", incomplete.ProjectID)
	require.Equal(t, "workspace.backend", incomplete.WorkspaceID)
}

func TestFindRootRejectsChildBridgeIdentityMismatch(t *testing.T) {
	_, child := newRootWithChildSubmodule(t)
	bridgePath := filepath.Join(child, ".harness", "bridge.yaml")
	require.NoError(t, os.WriteFile(bridgePath, []byte(`schema_version: 1
project_id: project.other
root_remote: https://example.invalid/root.git
workspace_id: workspace.backend
discovery: git-superproject
contract_fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
commands_path: .harness/commands.yaml
`), 0o600))

	_, err := FindRoot(context.Background(), child)

	require.ErrorContains(t, err, "project ID")
}

func TestFindRootSelectsDirectoryWorkspaceFromStartPath(t *testing.T) {
	root := t.TempDir()
	frontend := filepath.Join(root, "frontend")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o755))
	require.NoError(t, os.MkdirAll(frontend, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(`schema_version: 1
project_id: project.example
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

	got, err := FindRoot(context.Background(), frontend)

	require.NoError(t, err)
	require.Equal(t, "workspace.frontend", got.CurrentWorkspaceID)
}

func newRootWithChildSubmodule(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	childSource := filepath.Join(parent, "backend-source")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.MkdirAll(childSource, 0o755))

	git(t, childSource, "init", "-b", "main")
	git(t, childSource, "config", "user.name", "Test User")
	git(t, childSource, "config", "user.email", "test@example.com")
	require.NoError(t, os.MkdirAll(filepath.Join(childSource, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(childSource, ".harness", "bridge.yaml"), []byte(`schema_version: 1
project_id: project.example
root_remote: https://example.invalid/root.git
workspace_id: workspace.backend
discovery: git-superproject
contract_fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
commands_path: .harness/commands.yaml
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(childSource, "README.md"), []byte("# backend\n"), 0o600))
	git(t, childSource, "add", ".")
	git(t, childSource, "commit", "-m", "chore: initialize backend")

	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(`schema_version: 1
project_id: project.example
root_remote: https://example.invalid/root.git
workspaces:
  - id: workspace.root
    kind: root
    path: .
    responsibilities: [orchestration]
    dependencies: []
  - id: workspace.backend
    kind: submodule
    path: backend
    remote: https://example.invalid/backend.git
    responsibilities: [backend]
    dependencies: [workspace.root]
    contract_fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    commands_path: .harness/commands.yaml
`), 0o600))
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "chore: initialize orchestration")
	git(t, root, "-c", "protocol.file.allow=always", "submodule", "add", childSource, "backend")
	git(t, root, "commit", "-am", "chore: add backend workspace")

	resolvedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	resolvedChild, err := filepath.EvalSymlinks(filepath.Join(root, "backend"))
	require.NoError(t, err)
	return resolvedRoot, resolvedChild
}

func git(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
