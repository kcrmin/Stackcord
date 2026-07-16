package context_test

import (
	stdcontext "context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRefreshMarksGeneratedSummaryStaleWhenSourceChanges(t *testing.T) {
	root := contextFixture(t, false)
	policyPath := filepath.Join(root, "specs", "policies", "account.md")
	data, err := os.ReadFile(policyPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(policyPath, []byte(strings.ReplaceAll(string(data), "limit: 6", "limit: 7")), 0o600))

	got, issues := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, errorsOnly(issues))
	require.Contains(t, got.Stale, "docs.generated.current")
	require.Contains(t, got.Stale, "scenario.account.rate-limited")
}

func TestRefreshDoesNotLetTaskTitleOverrideApprovedPolicy(t *testing.T) {
	root := contextFixture(t, true)
	got, _ := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Equal(t, "specs/policies/account.md", got.Index["policy.account.rate-limit"].Path)
	require.Contains(t, got.Unknown, "work.GH-12.semantic-conflict")
}

func TestRefreshReadOnlyDoesNotWriteCheckpoint(t *testing.T) {
	root := contextFixture(t, false)
	indexPath := filepath.Join(root, ".harness", "state", "context-index.json")
	before, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	_, _ = contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	after, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func contextFixture(t *testing.T, semanticConflict bool) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{
		filepath.Join(root, ".harness", "state"),
		filepath.Join(root, ".harness", "work", "items"),
		filepath.Join(root, "specs", "policies"),
		filepath.Join(root, "specs", "scenarios"),
		filepath.Join(root, "docs", "generated"),
	} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))

	policy := "---\nschema_version: 1\nid: policy.account.rate-limit\nkind: policy\nstatus: approved\nrevision: 1\nrefs: []\n---\nlimit: 6\n"
	policyFingerprint, err := contextpkg.Fingerprint("markdown", []byte(policy))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "account.md"), []byte(policy), 0o600))

	scenario := "---\nschema_version: 1\nid: scenario.account.rate-limited\nkind: scenario\nstatus: approved\nrevision: 1\nrefs: [policy.account.rate-limit]\n---\nThe seventh request is rejected.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "scenarios", "rate-limited.md"), []byte(scenario), 0o600))

	generated := "---\nschema_version: 1\nid: docs.generated.current\nkind: decision\nstatus: approved\nrevision: 1\nrefs: []\nsources:\n  - source: policy.account.rate-limit\n    fingerprint: " + policyFingerprint + "\n---\nCurrent account policy summary.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "generated", "current.md"), []byte(generated), 0o600))

	checkpoint := map[string]any{"schema_version": 1, "index": map[string]any{}}
	checkpointData, err := json.MarshalIndent(checkpoint, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "state", "context-index.json"), append(checkpointData, '\n'), 0o600))

	if semanticConflict {
		work := "schema_version: 1\nid: work.GH-12\nsemantic_overrides: [policy.account.rate-limit]\n"
		require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "items", "GH-12.yaml"), []byte(work), 0o600))
	}
	return root
}

func errorsOnly(items []domain.Item) []domain.Item {
	result := make([]domain.Item, 0)
	for _, item := range items {
		if strings.HasPrefix(item.Code, "context.error") {
			result = append(result, item)
		}
	}
	return result
}
