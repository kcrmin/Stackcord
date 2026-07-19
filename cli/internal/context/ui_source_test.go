package context_test

import (
	stdcontext "context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCanonicalExternalUIChangeMarksMappedContextStale(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, ".harness", "sources", "ui"), filepath.Join(root, "specs", "ui"), filepath.Join(root, "docs", "generated")} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.ui-source\nlocale: en\n"), 0o600))
	registration := "schema_version: 1\nid: ui.external.refund\nkind: mockup\nauthority: canonical\nsource: archive\nsource_version: design-2\nlicense: MIT\ncontent_hash: sha256:" + strings.Repeat("b", 64) + "\nbaseline_fingerprint: sha256:" + strings.Repeat("a", 64) + "\nfetched_at: 2026-07-18T00:00:00Z\nmapped_refs: [ui.refund]\nconsumers: [workspace.frontend]\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "sources", "ui", "ui.external.refund.yaml"), []byte(registration), 0o600))
	ui := "---\nschema_version: 1\nid: ui.refund\nkind: ui\nstatus: approved\nrevision: 1\nrefs: []\n---\n\nRefund UI.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "ui", "refund.md"), []byte(ui), 0o600))

	snapshot, issues := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, errorsOnly(issues))
	for _, id := range []string{"ui.external.refund", "ui.refund", "workspace.frontend"} {
		require.Contains(t, snapshot.Stale, id)
	}
	require.Contains(t, snapshot.Unknown, "ui.external.refund.reconciliation-required")
}

func TestRefreshRejectsSymlinkedExternalUISourceDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "sources"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.ui-source\nlocale: en\n"), 0o600))
	if err := os.Symlink(outside, filepath.Join(root, ".harness", "sources", "ui")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	_, issues := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Contains(t, issueCodes(issues), "context.error.ui-sources")
}

func issueCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
