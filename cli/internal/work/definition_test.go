package work

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestReadyDefinitionRequiresAcceptanceScopeOrderAndFirstTest(t *testing.T) {
	definition := validDefinition()
	definition.Acceptance = nil
	definition.MergeOrder = nil
	definition.FirstFailingTest = ""

	issues := ValidateDefinition(definition)

	require.Contains(t, codes(issues), "work.acceptance-required")
	require.Contains(t, codes(issues), "work.merge-order-required")
	require.Contains(t, codes(issues), "work.first-test-required")
}

func TestDefinitionFingerprintChangesWhenSemanticScopeExpands(t *testing.T) {
	before := validDefinition()
	after := before
	after.Scope.DBEntities = append([]string(nil), before.Scope.DBEntities...)
	after.Scope.DBEntities = append(after.Scope.DBEntities, "account_recovery")

	require.NotEqual(t, Fingerprint(before), Fingerprint(after))
}

func TestPlanDefinitionRejectsMissingCanonicalReference(t *testing.T) {
	root := definitionRoot(t)
	definition := validDefinition()
	definition.Refs = append(definition.Refs, "policy.missing")

	plan, err := PlanDefinition(context.Background(), root, definition)

	require.NoError(t, err)
	require.Contains(t, codes(plan.Blockers), "work.ref-missing")
}

func validDefinition() Definition {
	return Definition{
		SchemaVersion: 1,
		ID:            "work.account-recovery",
		Readiness:     Ready,
		Title:         "Recover an account safely",
		Outcome:       "An eligible user regains access without weakening proof requirements.",
		Acceptance: []AcceptanceScenario{{
			ID: "scenario.account-recovery", Given: "an eligible locked account", When: "the user completes recovery proof", Then: "access is restored", Failure: "invalid proof is rejected without account disclosure",
		}},
		Refs:       []string{"policy.account-recovery"},
		Workspaces: []string{"workspace.backend", "workspace.frontend", "workspace.root"},
		Scope: Scope{
			Repositories: []string{"repository.root", "repository.backend", "repository.frontend"},
			Paths:        []string{"backend/recovery", "frontend/recovery"},
			PolicyIDs:    []string{"policy.account-recovery"},
			ScenarioIDs:  []string{"scenario.account-recovery"},
			ContractIDs:  []string{"contract.account-recovery"},
			DBEntities:   []string{"account"},
			UIFlows:      []string{"ui.account-recovery"},
			RootPointers: []string{"workspace.backend", "workspace.frontend"},
		},
		Dependencies:     []string{},
		MergeOrder:       []string{"workspace.backend", "workspace.frontend", "workspace.root"},
		FirstFailingTest: "test.account-recovery-contract",
		Evidence: EvidenceRequirements{
			Kinds:               []string{"contract", "unit", "integration", "ui"},
			IntegrationRequired: true,
		},
	}
}

func definitionRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, ".harness", "work", "definitions"), filepath.Join(root, "specs", "policies"), filepath.Join(root, "specs", "scenarios"), filepath.Join(root, "contracts", "behaviors")} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n  - id: workspace.backend\n    kind: directory\n    path: backend\n    responsibilities: [backend]\n    dependencies: [workspace.root]\n  - id: workspace.frontend\n    kind: directory\n    path: frontend\n    responsibilities: [frontend]\n    dependencies: [workspace.backend]\n"), 0o600))
	documents := map[string]string{
		filepath.Join("specs", "policies", "recovery.md"):      document("policy.account-recovery", "policy"),
		filepath.Join("specs", "scenarios", "recovery.md"):     document("scenario.account-recovery", "scenario"),
		filepath.Join("contracts", "behaviors", "recovery.md"): document("contract.account-recovery", "contract"),
	}
	for path, content := range documents {
		require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte(content), 0o600))
	}
	return root
}

func document(id, kind string) string {
	return "---\nschema_version: 1\nid: " + id + "\nkind: " + kind + "\nstatus: approved\nrevision: 1\nrefs: []\n---\nDefined.\n"
}

func codes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
