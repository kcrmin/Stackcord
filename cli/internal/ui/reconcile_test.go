package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uiimport "fullstack-orchestrator/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestCanonicalUIChangeMarksMappedFlowsStale(t *testing.T) {
	current := uiimport.Registration{SchemaVersion: 1, ID: "ui.external.refund", Kind: "mockup", Authority: "canonical", ContentHash: uiDigest("a"), FetchedAt: time.Now().Add(-time.Hour), MappedRefs: []string{"ui.refund"}, Consumers: []string{"workspace.frontend"}}
	next := current
	next.ContentHash = uiDigest("b")
	next.FetchedAt = time.Now()

	state := uiimport.Reconcile(current, next)
	require.ElementsMatch(t, []string{"ui.refund", "workspace.frontend"}, state.StaleRefs)
	require.True(t, state.RequiresApproval)
}

func TestLoadRegistrationRejectsSymlinkedSourceDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "sources"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "ui.external.reference.yaml"), []byte("untrusted"), 0o600))
	if err := os.Symlink(outside, filepath.Join(root, ".harness", "sources", "ui")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	_, err := uiimport.LoadRegistration(root, "ui.external.reference")
	require.ErrorContains(t, err, "symlink")
}

func TestReferenceChangeHasNoAuthorityOverCanonicalUI(t *testing.T) {
	current := uiimport.Registration{SchemaVersion: 1, ID: "ui.external.reference", Kind: "mockup", Authority: "reference", ContentHash: uiDigest("a"), FetchedAt: time.Now().Add(-time.Hour), MappedRefs: []string{"ui.refund"}}
	next := current
	next.ContentHash = uiDigest("b")
	next.FetchedAt = time.Now()

	state := uiimport.Reconcile(current, next)
	require.Empty(t, state.StaleRefs)
	require.True(t, state.RequiresApproval, "updating source registration still needs review")
}

func TestUIReconcileRejectsAuthorityEscalation(t *testing.T) {
	current := uiimport.Registration{SchemaVersion: 1, ID: "ui.external.reference", Kind: "mockup", Authority: "reference", ContentHash: uiDigest("a"), FetchedAt: time.Now()}
	next := current
	next.Authority = "canonical"
	next.ContentHash = uiDigest("b")

	state := uiimport.Reconcile(current, next)
	require.Contains(t, state.Blockers, "ui.authority-change")
}

func uiDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
