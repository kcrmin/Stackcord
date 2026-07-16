package output_test

import (
	"bytes"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/output"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONUsesStableEnvelope(t *testing.T) {
	result := domain.Result{
		SchemaVersion: "1.0",
		ToolVersion:   "1.0.0",
		Command:       "doctor",
		OperationID:   "01JTEST",
		Status:        domain.StatusPassed,
		ExitCode:      0,
		Summary:       "Environment is ready.",
	}

	var out bytes.Buffer
	require.NoError(t, output.WriteJSON(&out, result))
	require.JSONEq(t, `{
		"schema_version":"1.0",
		"tool_version":"1.0.0",
		"command":"doctor",
		"operation_id":"01JTEST",
		"status":"passed",
		"exit_code":0,
		"summary":"Environment is ready.",
		"facts":[],
		"warnings":[],
		"blockers":[],
		"changes":[],
		"evidence":[],
		"next_actions":[],
		"approval":{"required":false,"class":"A","reason":""},
		"timing_ms":0
	}`, out.String())
}
