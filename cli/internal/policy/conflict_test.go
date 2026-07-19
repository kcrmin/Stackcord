package policy_test

import (
	"testing"
	"time"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/policy"
	"github.com/stretchr/testify/require"
)

func TestConflictMatrix(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	base := policy.Claim{ID: "claim.active", Repository: "root", Workspace: "workspace.identity", Owner: "alex", ExpiresAt: now.Add(time.Hour), Observable: true}
	cases := []struct {
		name      string
		candidate policy.Candidate
		claim     policy.Claim
		want      policy.ConflictLevel
	}{
		{"independent", policy.Candidate{Repository: "root", Workspace: "workspace.web", Paths: []string{"apps/web/**"}}, with(base, func(c *policy.Claim) { c.Paths = []string{"services/identity/**"} }), policy.ConflictClear},
		{"path overlap", policy.Candidate{Repository: "root", Paths: []string{"services/identity/**"}}, with(base, func(c *policy.Claim) { c.Paths = []string{"services/identity/handler/**"} }), policy.ConflictCoordinate},
		{"same policy different files", policy.Candidate{Repository: "root", PolicyIDs: []string{"policy.account.recovery"}}, with(base, func(c *policy.Claim) { c.PolicyIDs = []string{"policy.account.recovery"} }), policy.ConflictBlock},
		{"same contract", policy.Candidate{Repository: "root", ContractIDs: []string{"contract.identity.recovery.v1"}}, with(base, func(c *policy.Claim) { c.ContractIDs = []string{"contract.identity.recovery.v1"} }), policy.ConflictBlock},
		{"same database entity", policy.Candidate{Repository: "root", DBEntities: []string{"identity.recovery_token"}}, with(base, func(c *policy.Claim) { c.DBEntities = []string{"identity.recovery_token"} }), policy.ConflictCoordinate},
		{"same migration slot", policy.Candidate{Repository: "root", MigrationSlots: []string{"identity:20260716-01"}}, with(base, func(c *policy.Claim) { c.MigrationSlots = []string{"identity:20260716-01"} }), policy.ConflictBlock},
		{"same UI flow", policy.Candidate{Repository: "root", UIFlows: []string{"flow.account.recovery"}}, with(base, func(c *policy.Claim) { c.UIFlows = []string{"flow.account.recovery"} }), policy.ConflictCoordinate},
		{"dependency major", policy.Candidate{Repository: "root", DependencyMajors: []string{"shared.auth@3"}}, with(base, func(c *policy.Claim) { c.DependencyMajors = []string{"shared.auth@3"} }), policy.ConflictCoordinate},
		{"same stable product meaning", policy.Candidate{Repository: "root", StableIDs: []string{"feature.account-recovery"}}, with(base, func(c *policy.Claim) { c.StableIDs = []string{"feature.account-recovery"} }), policy.ConflictBlock},
		{"root pointer order", policy.Candidate{Repository: "root", RootPointer: true}, with(base, func(c *policy.Claim) { c.RootPointer = true }), policy.ConflictCoordinate},
		{"expired claim", policy.Candidate{Repository: "root", Paths: []string{"same/**"}}, with(base, func(c *policy.Claim) { c.Paths = []string{"same/**"}; c.ExpiresAt = now.Add(-time.Minute) }), policy.ConflictClear},
		{"unobservable provider", policy.Candidate{Repository: "root", Paths: []string{"same/**"}}, with(base, func(c *policy.Claim) { c.Paths = []string{"same/**"}; c.Observable = false }), policy.ConflictUnknown},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.candidate.Now = now
			snapshot := contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{"contract.identity.recovery.v1": {ID: "contract.identity.recovery.v1", Kind: "behavior", ContractRegistered: true}}}
			report := policy.CheckConflict(test.candidate, []policy.Claim{test.claim}, snapshot)
			require.Equal(t, test.want, report.Level)
			if report.Level != policy.ConflictClear {
				require.NotEmpty(t, report.NextAction)
			}
		})
	}
}

func TestContractConflictUsesRegistryKindAndConsumerImpact(t *testing.T) {
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	claim := policy.Claim{ID: "claim.contract", Repository: "root", Owner: "alex", Observable: true, ExpiresAt: now.Add(time.Hour), ContractIDs: []string{"contract.interface.refund"}}
	snapshot := contextpkg.Snapshot{
		Index: map[string]contextpkg.IndexEntry{
			"contract.interface.refund": {ID: "contract.interface.refund", Kind: "interface", ContractRegistered: true},
			"contract.business.refund":  {ID: "contract.business.refund", Kind: "business", ContractRegistered: true},
		},
		Impact: map[string][]string{"contract.interface.refund": {"ui.refund"}},
	}

	interfaceOverlap := policy.CheckConflict(policy.Candidate{Repository: "root", ContractIDs: []string{"contract.interface.refund"}, Now: now}, []policy.Claim{claim}, snapshot)
	require.Equal(t, policy.ConflictCoordinate, interfaceOverlap.Level)

	consumerOverlap := policy.CheckConflict(policy.Candidate{Repository: "root", StableIDs: []string{"ui.refund"}, Now: now}, []policy.Claim{claim}, snapshot)
	require.Equal(t, policy.ConflictCoordinate, consumerOverlap.Level)

	claim.ContractIDs = []string{"contract.business.refund"}
	businessOverlap := policy.CheckConflict(policy.Candidate{Repository: "root", ContractIDs: []string{"contract.business.refund"}, Now: now}, []policy.Claim{claim}, snapshot)
	require.Equal(t, policy.ConflictBlock, businessOverlap.Level)
}

func TestMissingOrStaleContractRegistryIsUnknown(t *testing.T) {
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	claim := policy.Claim{ID: "claim.contract", Repository: "root", Owner: "alex", Observable: true, ExpiresAt: now.Add(time.Hour), ContractIDs: []string{"contract.business.refund"}}
	candidate := policy.Candidate{Repository: "root", ContractIDs: []string{"contract.business.refund"}, Now: now}

	missing := policy.CheckConflict(candidate, []policy.Claim{claim}, contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{}})
	require.Equal(t, policy.ConflictUnknown, missing.Level)

	stale := policy.CheckConflict(candidate, []policy.Claim{claim}, contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{"contract.business.refund": {ID: "contract.business.refund", Kind: "business", ContractRegistered: true}}, Stale: []string{"contract.business.refund"}})
	require.Equal(t, policy.ConflictUnknown, stale.Level)
}

func with(claim policy.Claim, mutate func(*policy.Claim)) policy.Claim {
	mutate(&claim)
	return claim
}
