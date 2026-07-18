package integration_test

import (
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/integration"
	"fullstack-orchestrator/cli/internal/work"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRequiresContractProviderConsumerAndRootPointerOrder(t *testing.T) {
	definition := integrationDefinition()
	plan := integration.Plan(
		[]work.Definition{definition},
		[]integration.ProviderState{{WorkID: definition.ID, Status: "review", Revision: "provider-r1", DefinitionFingerprint: definition.Fingerprint, Confirmed: true}},
		integrationWorkspaceStates(),
	)

	require.Empty(t, plan.Blockers)
	kinds := make([]integration.StepKind, 0, len(plan.Steps))
	refs := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		kinds = append(kinds, step.Kind)
		refs = append(refs, step.Ref)
	}
	require.Equal(t, []integration.StepKind{integration.ContractStep, integration.WorkspaceStep, integration.WorkspaceStep, integration.RootPointerStep}, kinds)
	require.Equal(t, []string{"contract.interface.accounts", "workspace.backend", "workspace.frontend", "workspace.backend"}, refs)
}

func TestIntegrationBlocksUnknownProviderAndPointerOwnershipOverlap(t *testing.T) {
	definition := integrationDefinition()
	second := definition
	second.ID = "work.accounts-admin"
	second.Fingerprint = integrationDigest("b")
	second.Scope.RootPointers = []string{"workspace.backend"}
	plan := integration.Plan(
		[]work.Definition{definition, second},
		[]integration.ProviderState{{WorkID: definition.ID, Confirmed: false}, {WorkID: second.ID, Status: "review", Revision: "provider-r2", DefinitionFingerprint: second.Fingerprint, Confirmed: true}},
		integrationWorkspaceStates(),
	)

	require.Contains(t, integrationCodes(plan.Blockers), "integrate.provider-unknown")
	require.Contains(t, integrationCodes(plan.Blockers), "integrate.pointer-overlap")
}

func TestVerifyIntegrationBindsEveryStepToExactCurrentIdentity(t *testing.T) {
	definition := integrationDefinition()
	workspaces := integrationWorkspaceStates()
	plan := integration.Plan(
		[]work.Definition{definition},
		[]integration.ProviderState{{WorkID: definition.ID, Status: "review", Revision: "provider-r1", DefinitionFingerprint: definition.Fingerprint, Confirmed: true}},
		workspaces,
	)
	require.Empty(t, plan.Blockers)
	evidence := make([]integration.Evidence, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		evidence = append(evidence, integration.Evidence{
			StepID: step.ID, WorkID: step.WorkID, Kind: step.RequiredEvidence, WorkspaceID: step.WorkspaceID,
			DefinitionFingerprint: step.DefinitionFingerprint, ContractFingerprint: integrationDigest("c"), ProviderRevision: step.ProviderRevision,
			Commit: step.Commit, Digest: integrationDigest("e"),
		})
	}

	result := integration.Verify(plan, evidence, workspaces, integrationDigest("c"))
	require.Equal(t, domain.StatusPassed, result.Status)

	workspaces[1].Commit = strings.Repeat("f", 40)
	changed := integration.Verify(plan, evidence, workspaces, integrationDigest("c"))
	require.Equal(t, domain.StatusBlocked, changed.Status)
	require.Contains(t, integrationCodes(changed.Blockers), "integrate.workspace-changed")
}

func TestVerifyIntegrationRejectsTamperedPlanShape(t *testing.T) {
	definition := integrationDefinition()
	workspaces := integrationWorkspaceStates()
	plan := integration.Plan([]work.Definition{definition}, []integration.ProviderState{{WorkID: definition.ID, Status: "review", Revision: "provider-r1", DefinitionFingerprint: definition.Fingerprint, Confirmed: true}}, workspaces)
	require.NotEmpty(t, plan.Steps)
	plan.Steps[0].Commit = "HEAD"

	result := integration.Verify(plan, nil, workspaces, integrationDigest("c"))
	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, integrationCodes(result.Blockers), "integrate.plan-invalid")
}

func integrationDefinition() work.Definition {
	return work.Definition{
		SchemaVersion: 1, ID: "work.accounts-recovery", Readiness: work.Ready, Fingerprint: integrationDigest("a"),
		Workspaces: []string{"workspace.backend", "workspace.frontend"}, MergeOrder: []string{"workspace.backend", "workspace.frontend"},
		Scope:    work.Scope{ContractIDs: []string{"contract.interface.accounts"}, RootPointers: []string{"workspace.backend"}},
		Evidence: work.EvidenceRequirements{IntegrationRequired: true},
	}
}

func integrationWorkspaceStates() []integration.WorkspaceState {
	return []integration.WorkspaceState{
		{ID: "workspace.root", Kind: "root", Commit: strings.Repeat("1", 40), Remote: "https://example.test/root.git", Clean: true, Published: true},
		{ID: "workspace.backend", Kind: "submodule", Commit: strings.Repeat("2", 40), Remote: "https://example.test/backend.git", Clean: true, Published: true, ExpectedPointer: strings.Repeat("2", 40), ActualPointer: strings.Repeat("2", 40)},
		{ID: "workspace.frontend", Kind: "submodule", Commit: strings.Repeat("3", 40), Remote: "https://example.test/frontend.git", Clean: true, Published: true, ExpectedPointer: strings.Repeat("3", 40), ActualPointer: strings.Repeat("3", 40)},
	}
}

func integrationCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}

func integrationDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
