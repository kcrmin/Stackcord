package project_test

import (
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/project"
	"github.com/stretchr/testify/require"
)

func TestStartWorkCreatesClaimAndBranchCheckpoint(t *testing.T) {
	request := project.StartWorkRequest{
		Root: t.TempDir(), WorkID: "work.01JACCOUNT", ClaimID: "claim.01JACCOUNT",
		Owner: "alex", Branch: "feature/GH-142-account-recovery",
		ExpiresAt: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
		Candidate: policy.Candidate{Repository: "root", Workspace: "workspace.identity", ContractIDs: []string{"contract.identity.recovery.v1"}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)},
	}
	plan := project.StartWork(request)
	require.Empty(t, plan.Blockers)
	require.Len(t, plan.Files, 2)
	require.Equal(t, ".harness/work/claims/claim.01JACCOUNT.yaml", plan.Files[0].Path)
	require.Contains(t, string(plan.Files[0].Content), "contract.identity.recovery.v1")
}

func TestStartWorkBlocksSharedContractConflict(t *testing.T) {
	request := project.StartWorkRequest{
		Root: t.TempDir(), WorkID: "work.new", ClaimID: "claim.new", Owner: "sam", Branch: "feature/shared-change",
		ExpiresAt: time.Now().Add(time.Hour), Candidate: policy.Candidate{Repository: "root", ContractIDs: []string{"contract.shared.v1"}, Now: time.Now()},
		ActiveClaims: []policy.Claim{{ID: "claim.existing", Repository: "root", ContractIDs: []string{"contract.shared.v1"}, Observable: true, ExpiresAt: time.Now().Add(time.Hour)}},
	}
	plan := project.StartWork(request)
	require.Empty(t, plan.Files)
	require.NotEmpty(t, plan.Blockers)
	require.Equal(t, "conflict.contract", plan.Blockers[0].Code)
}

func TestFinishWorkRequiresVerificationEvidence(t *testing.T) {
	result := project.FinishWork(project.FinishWorkRequest{WorkID: "work.example"})
	require.Equal(t, domain.StatusBlocked, result.Status)

	result = project.FinishWork(project.FinishWorkRequest{WorkID: "work.example", Evidence: []string{"evidence.tdd", "evidence.integration"}})
	require.Equal(t, domain.StatusPassed, result.Status)
}
