package ui_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	uiimport "fullstack-orchestrator/cli/internal/ui"
	"fullstack-orchestrator/cli/internal/work"
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

func TestUIIntegrationAcceptsBaselineOnlyInsideExecutableWorkScope(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "sources", "ui"), 0o700))
	registration := uiimport.Registration{SchemaVersion: 1, ID: "ui.external.refund", Kind: "mockup", Authority: "canonical", Source: "archive", SourceVersion: "design-2", License: "MIT", ContentHash: uiDigest("b"), BaselineFingerprint: uiDigest("a"), FetchedAt: time.Now(), MappedRefs: []string{"ui.refund"}, Consumers: []string{"workspace.frontend"}}
	data := "schema_version: 1\nid: ui.external.refund\nkind: mockup\nauthority: canonical\nsource: archive\nsource_version: design-2\nlicense: MIT\ncontent_hash: " + registration.ContentHash + "\nbaseline_fingerprint: " + registration.BaselineFingerprint + "\nfetched_at: " + registration.FetchedAt.UTC().Format(time.RFC3339) + "\nmapped_refs: [ui.refund]\nconsumers: [workspace.frontend]\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "sources", "ui", "ui.external.refund.yaml"), []byte(data), 0o600))
	definition := work.Definition{ID: "work.refund-ui", Readiness: work.Ready, Workspaces: []string{"workspace.frontend"}, Scope: work.Scope{UIFlows: []string{"ui.refund"}}, Evidence: work.EvidenceRequirements{IntegrationRequired: true}}

	accepted, plan, err := uiimport.AcceptIntegratedBaseline(root, registration.ID, definition)
	require.NoError(t, err)
	require.Equal(t, registration.ContentHash, accepted.BaselineFingerprint)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	loaded, err := uiimport.LoadRegistration(root, registration.ID)
	require.NoError(t, err)
	require.Equal(t, loaded.ContentHash, loaded.BaselineFingerprint)

	definition.Scope.UIFlows = nil
	_, _, err = uiimport.AcceptIntegratedBaseline(root, registration.ID, definition)
	require.ErrorContains(t, err, "scope")
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
