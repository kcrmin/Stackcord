package command_test

import (
	"bytes"
	"os"
	"path/filepath"
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
	require.JSONEq(t, `{
		"schema_version":"1.0",
		"tool_version":"1.0.0",
		"command":"doctor",
		"operation_id":"doctor-read-only",
		"status":"passed",
		"exit_code":0,
		"summary":"Environment inspection completed.",
		"facts":[],"warnings":[],"blockers":[],"changes":[],"evidence":[],"next_actions":[],
		"approval":{"required":false,"class":"A","reason":""},
		"timing_ms":0
	}`, stdout.String())
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
