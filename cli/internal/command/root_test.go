package command_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/command"
	"github.com/stretchr/testify/require"
)

func TestDoctorWritesStableJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &stderr)
	cmd.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, cmd.Execute())
	require.Empty(t, stderr.String())
	require.JSONEq(t, fmt.Sprintf(`{
		"schema_version":"1.0",
		"tool_version":"1.0.0",
		"command":"doctor",
		"operation_id":"doctor-read-only",
		"status":"passed",
		"exit_code":0,
		"summary":"Environment inspection completed.",
		"facts":[{"code":"environment.os","message":%q},{"code":"environment.arch","message":%q},{"code":"environment.go","message":%q}],"warnings":[],"blockers":[],"changes":[],"evidence":[],"next_actions":[],
		"approval":{"required":false,"class":"A","reason":""},
		"timing_ms":0
	}`, runtime.GOOS, runtime.GOARCH, runtime.Version()), stdout.String())
}

func TestContextAuditInspectsProjectWithoutWriting(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "state"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs", "policies"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	policy := "---\nschema_version: 1\nid: policy.example.ready\nkind: policy\nstatus: approved\nrevision: 1\nrefs: []\n---\nReady.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "ready.md"), []byte(policy), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &stderr)
	cmd.SetArgs([]string{"context", "audit", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"context.documents"`)
	_, err := os.Stat(filepath.Join(root, ".harness", "state", "context-index.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestProjectInitPlansThenAppliesNeutralHarness(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"project", "init", "--root", root, "--id", "project.command-example", "--name", "Command Example", "--locale", "en", "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "project.init.plan")
	_, err := os.Stat(filepath.Join(root, ".harness", "manifest.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	stdout.Reset()
	cmd = command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"project", "init", "--root", root, "--id", "project.command-example", "--name", "Command Example", "--locale", "en", "--apply", "--json"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))
	require.Contains(t, stdout.String(), "project.init")
}

func TestGitInspectCommandReportsActualState(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		process := exec.Command("git", args...)
		process.Dir = root
		output, err := process.CombinedOutput()
		require.NoError(t, err, string(output))
	}
	run("init", "--initial-branch=main")
	run("config", "user.email", "fixture@example.invalid")
	run("config", "user.name", "Fixture")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600))
	run("add", "README.md")
	run("commit", "-m", "chore: initialize")

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"git", "inspect", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `"git.branch"`)
	require.Contains(t, stdout.String(), `"main"`)
}

func TestCommandSurfaceCoversProjectLifecycle(t *testing.T) {
	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	paths := []string{
		"project draft", "project init", "project adopt",
		"context audit", "context refresh", "context pack",
		"git inspect", "git sync-plan", "git worktree-plan",
		"work next", "work conflict", "work start", "work finish", "work handoff",
		"change plan", "contract check", "contract impact",
		"db diff", "db diagram", "ui import", "integrate plan",
		"verify release", "rc create", "rc verify", "release prepare", "release publish",
	}
	for _, path := range paths {
		found, _, err := cmd.Find(strings.Fields(path))
		require.NoError(t, err, path)
		require.Equal(t, strings.Fields(path)[len(strings.Fields(path))-1], found.Name(), path)
	}
}

func TestDoctorExportsPrivacySafeDiagnostics(t *testing.T) {
	exportPath := filepath.Join(t.TempDir(), "diagnostic.zip")
	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--root", t.TempDir(), "--export", exportPath, "--json"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, exportPath)
	require.Contains(t, stdout.String(), "diagnostic.export")
}
