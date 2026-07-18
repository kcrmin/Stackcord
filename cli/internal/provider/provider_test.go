package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestReconcileRejectsCachedOrDriftedProviderState(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	expectation := validExpectation()
	mapping := validMapping()
	snapshot := validSnapshot(now)
	snapshot.FetchedAt = now.Add(-25 * time.Hour)
	snapshot.DefinitionFingerprint = "sha256:" + strings.Repeat("0", 64)
	snapshot.Source = "cache"

	state := Reconcile(expectation, mapping, snapshot, now)

	require.Equal(t, Unknown, state.Confidence)
	require.Contains(t, providerCodes(state.Issues), "provider.snapshot-stale")
	require.Contains(t, providerCodes(state.Issues), "provider.definition-drift")
	require.Contains(t, providerCodes(state.Issues), "provider.not-live")
}

func TestReconcileAcceptsFreshConnectorSnapshotWithExactRevision(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	state := Reconcile(validExpectation(), validMapping(), validSnapshot(now), now)

	require.Equal(t, Confirmed, state.Confidence)
	require.Empty(t, state.Issues)
	require.Equal(t, "in_progress", state.Status)
	require.Equal(t, "ryan", state.Owner)
}

func TestReconcileDetectsDependencyMappingDrift(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	snapshot := validSnapshot(now)
	snapshot.Dependencies = []string{"JIRA-999"}

	state := Reconcile(validExpectation(), validMapping(), snapshot, now)

	require.Equal(t, Unknown, state.Confidence)
	require.Contains(t, providerCodes(state.Issues), "provider.dependency-drift")
}

func TestCanonicalMappingLocationRejectsEscapedParentSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work"), 0o700))
	if err := os.Symlink(outside, filepath.Join(root, ".harness", "work", "mappings")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	target := filepath.Join(outside, "work.account-recovery.yaml")
	require.NoError(t, os.WriteFile(target, []byte("escaped\n"), 0o600))

	err := ValidateCanonicalMappingLocation(root, filepath.Join(root, ".harness", "work", "mappings", "work.account-recovery.yaml"))

	require.Error(t, err)
}

func validExpectation() Expectation {
	return Expectation{WorkID: "work.account-recovery", DefinitionFingerprint: "sha256:" + strings.Repeat("a", 64), Dependencies: []string{"work.account-contract"}}
}

func validMapping() Mapping {
	return Mapping{
		SchemaVersion:         1,
		WorkID:                "work.account-recovery",
		DefinitionFingerprint: "sha256:" + strings.Repeat("a", 64),
		Provider:              "jira",
		ItemID:                "JIRA-123",
		DependencyItems:       map[string]string{"work.account-contract": "JIRA-100"},
	}
}

func validSnapshot(now time.Time) Snapshot {
	return Snapshot{
		SchemaVersion:         1,
		Provider:              "jira",
		ItemID:                "JIRA-123",
		Revision:              "42",
		Status:                "in_progress",
		Owner:                 "ryan",
		Dependencies:          []string{"JIRA-100"},
		Capabilities:          Capabilities{Hierarchy: true, Dependencies: true, Claim: "verified", Revision: true},
		DefinitionFingerprint: "sha256:" + strings.Repeat("a", 64),
		FetchedAt:             now.Add(-time.Minute),
		Source:                "connector-live",
		RawHash:               "sha256:" + strings.Repeat("b", 64),
	}
}

func providerCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
