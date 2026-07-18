package ui_test

import (
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestUIBaselineIdentityIsSafeAndDeterministic(t *testing.T) {
	baseline := ui.Baseline{
		SchemaVersion:   1,
		ID:              "ui.baseline.checkout",
		WorkspaceID:     "workspace.ui",
		WorkspaceCommit: strings.Repeat("a", 40),
		WorkspaceRemote: "https://example.test/product-ui.git",
		SourceIDs:       []string{"ui.external.checkout"},
		MappedRefs:      []string{"ui.checkout"},
		Consumers:       []string{"workspace.frontend"},
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
