package context_test

import (
	stdcontext "context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"github.com/stretchr/testify/require"
)

func TestContractRegistryAddsTypedImpactAndPropagatesFingerprintDrift(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, ".harness"), filepath.Join(root, "contracts", "behaviors"), filepath.Join(root, "specs", "ui"), filepath.Join(root, "docs", "generated")} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.contract-impact\nlocale: en\n"), 0o600))
	source := []byte("---\nschema_version: 1\nid: contract.behavior.refund-timeout\nkind: behavior\nstatus: approved\nrevision: 1\nrefs: []\n---\n\nRefund timeout behavior.\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "behaviors", "refund-timeout.md"), source, 0o600))
	ui := "---\nschema_version: 1\nid: ui.refund\nkind: ui\nstatus: approved\nrevision: 1\nrefs: [contract.behavior.refund-timeout]\n---\n\nRefund UI.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "ui", "refund.md"), []byte(ui), 0o600))
	fingerprint := sha256.Sum256(source)
	registry := "schema_version: 1\ncontracts:\n  - id: contract.behavior.refund-timeout\n    kind: behavior\n    status: approved\n    revision: 1\n    source: behaviors/refund-timeout.md\n    compatibility: coordinated\n    providers: []\n    consumers: []\n    product_ids: []\n    scenario_ids: []\n    data_ids: [data.refund]\n    ui_ids: [ui.refund]\n    migration_ids: [migration.refund]\n    work_ids: [work.refund-ui]\n    test_ids: []\n    refs: []\n    fingerprint: sha256:" + hex.EncodeToString(fingerprint[:]) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte(registry), 0o600))

	snapshot, issues := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, errorsOnly(issues))
	require.ElementsMatch(t, []string{"data.refund", "migration.refund", "ui.refund", "work.refund-ui"}, snapshot.Impact["contract.behavior.refund-timeout"])

	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "behaviors", "refund-timeout.md"), append(source, []byte("Changed.\n")...), 0o600))
	stale, issues := contextpkg.Refresh(stdcontext.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, errorsOnly(issues))
	for _, id := range []string{"contract.behavior.refund-timeout", "data.refund", "migration.refund", "ui.refund", "work.refund-ui"} {
		require.Contains(t, stale.Stale, id)
	}
	require.Contains(t, stale.Unknown, "contract.behavior.refund-timeout.fingerprint-drift")
}
