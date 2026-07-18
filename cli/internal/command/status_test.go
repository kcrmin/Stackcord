package command_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/command"
	"github.com/stretchr/testify/require"
)

func TestStatusReportsCombinedProjectIdentityAndOneNextAction(t *testing.T) {
	root := statusFixture(t)

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"status", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())

	require.Contains(t, stdout.String(), `"project_id":"project.example"`)
	require.Contains(t, stdout.String(), `"current_workspace_id":"workspace.root"`)
	require.Contains(t, stdout.String(), `"next_actions":[{`)
}

func TestSessionStartHookReadsCWDAndNeverWritesProjectState(t *testing.T) {
	root := statusFixture(t)
	contextPath := filepath.Join(root, ".harness", "local", "context", "context-index.json")

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{"cwd":` + fmt.Sprintf("%q", root) + `,"hook_event_name":"SessionStart"}`))
	cmd.SetArgs([]string{"hook", "session-start"})
	require.NoError(t, cmd.Execute())

	require.Contains(t, stdout.String(), `"hookEventName":"SessionStart"`)
	require.NoFileExists(t, contextPath)
}

func statusFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: git-local\nlive_status_source: git-local\n"), 0o600))
	runStatusGit(t, root, "init", "-b", "main")
	runStatusGit(t, root, "config", "user.name", "Test User")
	runStatusGit(t, root, "config", "user.email", "test@example.com")
	runStatusGit(t, root, "add", ".")
	runStatusGit(t, root, "commit", "-m", "chore: initialize project")
	return root
}

func runStatusGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
