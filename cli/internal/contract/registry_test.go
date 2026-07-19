package contract_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/contract"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBusinessContractRequiresObservableRejectedAndFailureBehavior(t *testing.T) {
	definition := contract.Definition{ID: "contract.business.account-recovery", Kind: contract.Business}
	issues := contract.Check(definition)
	require.Contains(t, contractCodes(issues), "contract.rejection-behavior-required")
	require.Contains(t, contractCodes(issues), "contract.failure-behavior-required")
}

func TestContractImpactIncludesUIDataMigrationAndActiveWork(t *testing.T) {
	registry := contract.Registry{SchemaVersion: 1, Contracts: []contract.Entry{{
		ID: "contract.behavior.refund-timeout", Kind: contract.Behavior, Status: contract.Approved,
		Source: "behaviors/refund-timeout.md", Revision: 1, Compatibility: contract.Coordinated,
		Providers: []string{}, Consumers: []string{}, ProductIDs: []string{}, ScenarioIDs: []string{},
		DataIDs: []string{"data.refund"}, UIIDs: []string{"ui.refund"}, MigrationIDs: []string{"migration.refund"},
		WorkIDs: []string{"work.refund-ui"}, TestIDs: []string{}, Refs: []string{}, Fingerprint: digestContract("source"),
	}}}

	impact := contract.Impact(registry, "contract.behavior.refund-timeout")
	require.ElementsMatch(t, []string{"ui.refund", "data.refund", "migration.refund", "work.refund-ui"}, impact.Dependents)
}

func TestLoadRegistryRejectsStaleSourceFingerprint(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts", "business"), 0o700))
	source := []byte("---\nschema_version: 1\nid: contract.business.refund\nkind: business\nstatus: approved\nrevision: 1\nrefs: []\n---\n\nRefunds require an eligible payment.\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "business", "refund.md"), source, 0o600))
	registry := "schema_version: 1\ncontracts:\n  - id: contract.business.refund\n    kind: business\n    status: approved\n    revision: 1\n    source: business/refund.md\n    compatibility: coordinated\n    providers: []\n    consumers: []\n    product_ids: []\n    scenario_ids: []\n    data_ids: []\n    ui_ids: []\n    migration_ids: []\n    work_ids: []\n    test_ids: []\n    refs: []\n    fingerprint: " + digestContract("different") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte(registry), 0o600))

	_, err := contract.LoadRegistry(root)
	require.ErrorContains(t, err, "fingerprint")
}

func contractCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}

func digestContract(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}
