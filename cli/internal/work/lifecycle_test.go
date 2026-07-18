package work

import (
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/evidence"
	"github.com/stretchr/testify/require"
)

func TestParentCannotBecomeDoneBeforeChildrenIntegrated(t *testing.T) {
	live := LiveState{
		Status: InProgress, Owner: "owner-a", Revision: "42", Confirmed: true,
		Children: map[string]State{"work.account-api": Review}, RootPointersConfirmed: true,
	}

	result := Transition(parentLifecycleDefinition(), live, lifecycleEvidence(), Done)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, lifecycleCodes(result.Blockers), "work.children-not-integrated")
}

func TestReviewRequiresCurrentPassingImplementationEvidence(t *testing.T) {
	live := LiveState{Status: InProgress, Owner: "owner-a", Revision: "42", Confirmed: true, Children: map[string]State{}}

	result := Transition(parentLifecycleDefinition(), live, nil, Review)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, lifecycleCodes(result.Blockers), "work.implementation-evidence-required")
}

func parentLifecycleDefinition() Definition {
	definition := Definition{
		SchemaVersion: 1, ID: "work.account-recovery", Readiness: Ready, Title: "Account recovery", Outcome: "Users recover access safely.",
		Acceptance: []AcceptanceScenario{{ID: "scenario.account-recovery", Given: "a recoverable account", When: "recovery is requested", Then: "access is restored", Failure: "unsafe recovery is rejected"}},
		Refs:       []string{}, Workspaces: []string{"workspace.backend", "workspace.root"}, Scope: Scope{Repositories: []string{"repository.root"}, Paths: []string{"backend", "frontend"}, RootPointers: []string{"workspace.backend"}},
		Dependencies: []string{}, MergeOrder: []string{"workspace.backend", "workspace.root"}, FirstFailingTest: "test.account-recovery",
		Evidence: EvidenceRequirements{Kinds: []string{"test", "integration"}, IntegrationRequired: true, UserValidation: true},
	}
	definition.Fingerprint = Fingerprint(definition)
	return definition
}

func lifecycleEvidence() []evidence.Record {
	fingerprint := parentLifecycleDefinition().Fingerprint
	contract := "sha256:" + strings.Repeat("b", 64)
	commit := strings.Repeat("c", 40)
	return []evidence.Record{
		{SchemaVersion: 1, ID: "evidence.test", Kind: "test", WorkID: "work.account-recovery", WorkspaceID: "workspace.backend", ExitCode: 0, Commit: commit, DefinitionFingerprint: fingerprint, ContractFingerprint: contract, OutputDigest: "sha256:" + strings.Repeat("d", 64)},
		{SchemaVersion: 1, ID: "evidence.integration", Kind: "integration", WorkID: "work.account-recovery", WorkspaceID: "workspace.root", ExitCode: 0, Commit: commit, DefinitionFingerprint: fingerprint, ContractFingerprint: contract, OutputDigest: "sha256:" + strings.Repeat("e", 64)},
	}
}

func lifecycleCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
