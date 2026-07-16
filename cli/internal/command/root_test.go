package command_test

import (
	"bytes"
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
