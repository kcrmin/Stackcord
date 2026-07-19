package hook

import (
	"encoding/json"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/continuity"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRenderSessionStartUsesSupportedAdditionalContext(t *testing.T) {
	snapshot := continuity.Snapshot{
		ProjectID:            "project.example",
		CanonicalFingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Overall:              continuity.Blocked,
		Issues:               []domain.Item{{Code: "workspace.pointer-mismatch", Message: "secret detail", Refs: []string{"workspace.backend", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}},
		NextActions:          []domain.Item{{Code: "workspace.pointer-review", Message: "Review pointer", Refs: []string{"workspace.backend"}}},
	}

	data, err := Render("session-start", snapshot)
	require.NoError(t, err)

	var value map[string]any
	require.NoError(t, json.Unmarshal(data, &value))
	specific := value["hookSpecificOutput"].(map[string]any)
	require.Equal(t, "SessionStart", specific["hookEventName"])
	require.Contains(t, specific["additionalContext"], "workspace.pointer-mismatch")
	require.NotContains(t, string(data), "secret detail")
	require.NotContains(t, string(data), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
}

func TestRenderPostCompactUsesSupportedSystemMessageOnly(t *testing.T) {
	data, err := Render("post-compact", continuity.Snapshot{ProjectID: "project.example", Overall: continuity.Unknown})
	require.NoError(t, err)

	var value map[string]any
	require.NoError(t, json.Unmarshal(data, &value))
	require.Equal(t, true, value["continue"])
	require.Contains(t, value["systemMessage"], "stackcord status")
	require.NotContains(t, value, "hookSpecificOutput")
}
