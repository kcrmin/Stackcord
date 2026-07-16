package release_test

import (
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/release"
	"github.com/stretchr/testify/require"
)

func TestCandidateDetectsEveryImmutableInputChange(t *testing.T) {
	input := validInput()
	candidate, result := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, result.Status)
	require.NotEmpty(t, candidate.Digest)
	require.Equal(t, domain.StatusPassed, release.VerifyCandidate(candidate, input).Status)

	cases := map[string]func(*release.Input){
		"root_commit":            func(i *release.Input) { i.RootCommit = "bbbbbbbb" },
		"workspace_commits":      func(i *release.Input) { i.WorkspaceCommits["workspace.web"] = "bbbbbbbb" },
		"artifact_digests":       func(i *release.Input) { i.ArtifactDigests["cli-darwin-arm64"] = "sha256:bbbb" },
		"schema_versions":        func(i *release.Input) { i.SchemaVersions["harness"] = "2" },
		"adapter_versions":       func(i *release.Input) { i.AdapterVersions["github"] = "2" },
		"sbom_digest":            func(i *release.Input) { i.SBOMDigest = "sha256:bbbb" },
		"provenance_digest":      func(i *release.Input) { i.ProvenanceDigest = "sha256:bbbb" },
		"signature_digests":      func(i *release.Input) { i.SignatureDigests["cli-darwin-arm64"] = "sha256:bbbb" },
		"gate_receipts":          func(i *release.Input) { i.GateReceipts["tests"] = "receipt-b" },
		"docs_fingerprint":       func(i *release.Input) { i.DocsFingerprint = "sha256:bbbb" },
		"user_validation_digest": func(i *release.Input) { i.UserValidationDigest = "sha256:bbbb" },
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			changed := validInput()
			mutate(&changed)
			result := release.VerifyCandidate(candidate, changed)
			require.Equal(t, domain.StatusBlocked, result.Status)
			require.Equal(t, field, result.Blockers[0].Refs[0])
		})
	}
}

func TestCandidateRejectsTamperedManifestDigest(t *testing.T) {
	input := validInput()
	candidate, created := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, created.Status)

	candidate.Digest = "sha256:tampered"
	result := release.VerifyCandidate(candidate, input)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitVerification, result.ExitCode)
	require.Equal(t, "digest", result.Blockers[0].Refs[0])
}

func TestCandidateDetectsVersionAndSchemaChanges(t *testing.T) {
	input := validInput()
	candidate, _ := release.CreateCandidate(input)

	changedVersion := input
	changedVersion.Version = "1.0.1"
	require.Equal(t, "version", release.VerifyCandidate(candidate, changedVersion).Blockers[0].Refs[0])

	candidate.SchemaVersion = 2
	result := release.VerifyCandidate(candidate, input)
	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, "schema_version", result.Blockers[0].Refs[0])
}

func TestCandidateRequiresReleaseEvidenceIdentities(t *testing.T) {
	input := validInput()
	input.UserValidationDigest = ""

	_, result := release.CreateCandidate(input)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitVerification, result.ExitCode)
	require.Equal(t, "user_validation_digest", result.Blockers[0].Refs[0])
}

func TestEveryProductionGapBlocksRelease(t *testing.T) {
	cases := map[string]func(*release.Gates){
		"flaky required check":    func(g *release.Gates) { g.RequiredChecksStable = false },
		"manual critical check":   func(g *release.Gates) { g.CriticalChecksAutomated = false },
		"unsigned artifact":       func(g *release.Gates) { g.ArtifactsSigned = false },
		"migration rollback":      func(g *release.Gates) { g.MigrationRollbackVerified = false },
		"unsafe hook":             func(g *release.Gates) { g.HooksTrustedReadOnly = false },
		"missing macOS journey":   func(g *release.Gates) { g.MacOSJourneyVerified = false },
		"missing Windows journey": func(g *release.Gates) { g.WindowsJourneyVerified = false },
		"pluginless continuation": func(g *release.Gates) { g.PluginlessContinuation = false },
		"user digest mismatch":    func(g *release.Gates) { g.UserValidationMatches = false },
		"unowned warning":         func(g *release.Gates) { g.Warnings = []release.Warning{{Code: "release.warning"}} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			gates := validGates()
			mutate(&gates)
			result := release.Verify(gates)
			require.Equal(t, domain.StatusBlocked, result.Status)
			require.Equal(t, domain.ExitVerification, result.ExitCode)
		})
	}
}

func TestPublishAlwaysRequiresExactProductionApproval(t *testing.T) {
	candidate, _ := release.CreateCandidate(validInput())
	plan, result := release.PlanPublish(candidate, policy.Consent{})
	require.Equal(t, domain.StatusApprovalRequired, result.Status)
	require.Empty(t, plan.Commands)

	consent := policy.Consent{Approved: true, ExactDReceipt: true, Action: policy.PublishProduction, Objective: "publish 1.0.0", Repository: "product", Target: candidate.Digest, ExpiresAt: time.Now().Add(time.Hour)}
	plan, result = release.PlanPublish(candidate, consent)
	require.Equal(t, domain.StatusPassed, result.Status)
	require.NotEmpty(t, plan.Commands)
}

func TestPublishRejectsCandidateWhoseManifestWasTampered(t *testing.T) {
	candidate, _ := release.CreateCandidate(validInput())
	candidate.Digest = "sha256:tampered"
	consent := policy.Consent{Approved: true, ExactDReceipt: true, Action: policy.PublishProduction, Objective: "publish 1.0.0", Repository: "product", Target: candidate.Digest, ExpiresAt: time.Now().Add(time.Hour)}

	plan, result := release.PlanPublish(candidate, consent)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Empty(t, plan.Commands)
}

func TestPublishRejectsDigestValidCandidateWithMissingEvidence(t *testing.T) {
	forged, _ := release.CreateCandidate(validInput())
	forged.Input.UserValidationDigest = ""
	consent := policy.Consent{Approved: true, ExactDReceipt: true, Action: policy.PublishProduction, Objective: "publish 1.0.0", Repository: "product", Target: forged.Digest, ExpiresAt: time.Now().Add(time.Hour)}

	plan, result := release.PlanPublish(forged, consent)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Empty(t, plan.Commands)
	require.Contains(t, result.Blockers, domain.Item{Code: "release.evidence-required", Message: "Release evidence identity is required.", Refs: []string{"user_validation_digest"}})
}

func validInput() release.Input {
	return release.Input{Version: "1.0.0", RootCommit: "aaaaaaaa", WorkspaceCommits: map[string]string{"workspace.web": "aaaaaaaa"}, ArtifactDigests: map[string]string{"cli-darwin-arm64": "sha256:aaaa", "plugin": "sha256:cccc"}, SchemaVersions: map[string]string{"harness": "1", "result": "1.0"}, AdapterVersions: map[string]string{"github": "1"}, SBOMDigest: "sha256:aaaa", ProvenanceDigest: "sha256:aaaa", SignatureDigests: map[string]string{"cli-darwin-arm64": "sha256:aaaa"}, GateReceipts: map[string]string{"tests": "receipt-a", "security": "receipt-a"}, DocsFingerprint: "sha256:aaaa", UserValidationDigest: "sha256:aaaa", Gates: validGates()}
}

func validGates() release.Gates {
	return release.Gates{RequiredChecksStable: true, CriticalChecksAutomated: true, ArtifactsSigned: true, MigrationRollbackVerified: true, HooksTrustedReadOnly: true, MacOSJourneyVerified: true, WindowsJourneyVerified: true, PluginlessContinuation: true, UserValidationMatches: true, Warnings: []release.Warning{}}
}
