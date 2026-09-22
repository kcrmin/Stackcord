package project_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/harness/context"
	"github.com/kcrmin/Stackcord/cli/internal/harness/policy"
	"github.com/kcrmin/Stackcord/cli/internal/harness/project"
	"github.com/stretchr/testify/require"
)

func TestStartWorkCreatesClaimAndBranchCheckpoint(t *testing.T) {
	request := project.StartWorkRequest{
		Root: t.TempDir(), WorkID: "work.01JACCOUNT", ClaimID: "claim.01JACCOUNT",
		Owner: "alex", Branch: "feature/GH-142-account-recovery",
		ExpiresAt: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
		Candidate: policy.Candidate{Repository: "root", Workspace: "workspace.identity", ContractIDs: []string{"contract.identity.recovery.v1"}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)},
		Snapshot:  contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{"contract.identity.recovery.v1": {ID: "contract.identity.recovery.v1", Kind: "interface", Status: "approved", ContractRegistered: true}}},
	}
	plan := project.StartWork(request)
	require.Empty(t, plan.Blockers)
	require.Len(t, plan.Files, 2)
	require.Equal(t, ".harness/work/claims/claim.01JACCOUNT.yaml", plan.Files[0].Path)
	require.Contains(t, string(plan.Files[0].Content), "contract.identity.recovery.v1")
}

func TestStartWorkUsesRepositoryGitConvention(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "git-conventions.yaml"), []byte("schema_version: 1\nbranch:\n  format: \"{type}/{issue}-{description}\"\n  types: [feat, fix]\n"), 0o600))
	now := time.Now().UTC()

	plan := project.StartWork(project.StartWorkRequest{
		Root: root, WorkID: "work.custom-branch", ClaimID: "claim.custom-branch", Owner: "alex",
		Branch: "feat/OPS-42-account-recovery", ExpiresAt: now.Add(time.Hour),
		Candidate: policy.Candidate{Repository: "repository.root", Now: now},
		Snapshot: contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{}},
	})

	require.Empty(t, plan.Blockers)
	require.Len(t, plan.Files, 2)

	rejected := project.StartWork(project.StartWorkRequest{
		Root: root, WorkID: "work.wrong-branch", ClaimID: "claim.wrong-branch", Owner: "alex",
		Branch: "feature/account-recovery", ExpiresAt: now.Add(time.Hour),
		Candidate: policy.Candidate{Repository: "repository.root", Now: now},
		Snapshot: contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{}},
	})
	require.Equal(t, "work.branch-invalid", rejected.Blockers[0].Code)
	require.Contains(t, rejected.Blockers[0].Message, "{type}/{issue}-{description}")
}

func TestStartWorkBlocksSharedContractConflict(t *testing.T) {
	request := project.StartWorkRequest{
		Root: t.TempDir(), WorkID: "work.new", ClaimID: "claim.new", Owner: "sam", Branch: "feature/shared-change",
		ExpiresAt: time.Now().Add(time.Hour), Candidate: policy.Candidate{Repository: "root", ContractIDs: []string{"contract.shared.v1"}, Now: time.Now()},
		ActiveClaims: []policy.Claim{{ID: "claim.existing", Repository: "root", ContractIDs: []string{"contract.shared.v1"}, Observable: true, ExpiresAt: time.Now().Add(time.Hour)}},
		Snapshot:     contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{"contract.shared.v1": {ID: "contract.shared.v1", Kind: "behavior", Status: "approved", ContractRegistered: true}}},
	}
	plan := project.StartWork(request)
	require.Empty(t, plan.Files)
	require.NotEmpty(t, plan.Blockers)
	require.Equal(t, "conflict.contract", plan.Blockers[0].Code)
}

func TestStartWorkRejectsInvalidIdentityBranchAndLease(t *testing.T) {
	now := time.Now().UTC()
	plan := project.StartWork(project.StartWorkRequest{
		Root: t.TempDir(), WorkID: "../work", ClaimID: "claim.bad", Owner: "alex", Branch: "agent/generated-work", ExpiresAt: now,
		Candidate: policy.Candidate{Repository: "root", Now: now},
	})

	require.Empty(t, plan.Files)
	require.NotEmpty(t, plan.Blockers)
	require.Equal(t, "work.request-invalid", plan.Blockers[0].Code)
}
