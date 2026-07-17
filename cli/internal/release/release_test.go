package release_test

import (
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/release"
	"github.com/stretchr/testify/require"
)

func TestCoreCandidateNeedsNoStrictSupplyChainEvidence(t *testing.T) {
	input := validCoreInput()
	candidate, created := release.CreateCandidate(input)

	require.Equal(t, domain.StatusPassed, created.Status)
	require.NotEmpty(t, candidate.Digest)
	require.Nil(t, candidate.Input.StrictEvidence)
	require.Equal(t, domain.StatusPassed, release.VerifyCandidate(candidate, input, validUserValidation(candidate.Digest)).Status)
}

func TestStrictCandidateRequiresSupplyChainEvidence(t *testing.T) {
	input := validCoreInput()
	input.Profile = release.ProfileStrictRelease

	_, missing := release.CreateCandidate(input)
	require.Equal(t, domain.StatusBlocked, missing.Status)
	require.Equal(t, "strict_evidence", missing.Blockers[0].Refs[0])

	input.StrictEvidence = validStrictEvidence()
	candidate, created := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, created.Status)
	require.Equal(t, domain.StatusPassed, release.VerifyCandidate(candidate, input, validUserValidation(candidate.Digest)).Status)

	input.StrictEvidence.SignatureDigests = nil
	_, incomplete := release.CreateCandidate(input)
	require.Equal(t, domain.StatusBlocked, incomplete.Status)
	require.Equal(t, "signature_digests", incomplete.Blockers[0].Refs[0])
}

func TestCandidateDetectsEveryCoreIdentityChange(t *testing.T) {
	input := validCoreInput()
	candidate, created := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, created.Status)
	validation := validUserValidation(candidate.Digest)

	cases := map[string]func(*release.Input){
		"version":              func(i *release.Input) { i.Version = "1.0.1" },
		"root_commit":          func(i *release.Input) { i.RootCommit = strings.Repeat("b", 40) },
		"workspace_commits":    func(i *release.Input) { i.WorkspaceCommits["workspace.root"] = strings.Repeat("b", 40) },
		"artifact_digests":     func(i *release.Input) { i.ArtifactDigests["archive"] = digest("b") },
		"product_fingerprint":  func(i *release.Input) { i.ProductFingerprint = digest("c") },
		"docs_fingerprint":     func(i *release.Input) { i.DocsFingerprint = digest("d") },
		"contract_fingerprint": func(i *release.Input) { i.ContractFingerprint = digest("e") },
		"tdd_evidence":         func(i *release.Input) { i.TDDEvidence["tests"] = digest("f") },
		"integration_evidence": func(i *release.Input) { i.IntegrationEvidence["integration"] = digest("1") },
		"migration_required": func(i *release.Input) {
			i.MigrationRequired = true
			i.MigrationEvidence, i.RollbackEvidence = digest("2"), digest("3")
		},
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			changed := validCoreInput()
			mutate(&changed)
			result := release.VerifyCandidate(candidate, changed, validation)
			require.Equal(t, domain.StatusBlocked, result.Status)
			require.Equal(t, field, result.Blockers[0].Refs[0])
		})
	}
}

func TestCandidateRejectsTamperedManifestDigest(t *testing.T) {
	input := validCoreInput()
	candidate, created := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, created.Status)
	candidate.Digest = digest("f")

	result := release.VerifyCandidate(candidate, input, validUserValidation(candidate.Digest))

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitVerification, result.ExitCode)
	require.Equal(t, "digest", result.Blockers[0].Refs[0])
}

func TestMigrationEvidenceIsConditional(t *testing.T) {
	input := validCoreInput()
	_, withoutMigration := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, withoutMigration.Status)

	input.MigrationRequired = true
	_, missing := release.CreateCandidate(input)
	require.Equal(t, domain.StatusBlocked, missing.Status)
	require.Equal(t, []string{"migration_evidence", "rollback_evidence"}, []string{missing.Blockers[0].Refs[0], missing.Blockers[1].Refs[0]})

	input.MigrationEvidence, input.RollbackEvidence = digest("1"), digest("2")
	_, complete := release.CreateCandidate(input)
	require.Equal(t, domain.StatusPassed, complete.Status)
}

func TestUserValidationMustReferenceTheSameCandidate(t *testing.T) {
	input := validCoreInput()
	candidate, _ := release.CreateCandidate(input)

	cases := map[string]func(*release.UserValidation){
		"candidate_digest": func(v *release.UserValidation) { v.CandidateDigest = digest("f") },
		"confirmed":        func(v *release.UserValidation) { v.Confirmed = false },
		"evidence_digest":  func(v *release.UserValidation) { v.EvidenceDigest = "" },
		"verified_at":      func(v *release.UserValidation) { v.VerifiedAt = "not-a-time" },
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			validation := validUserValidation(candidate.Digest)
			mutate(&validation)
			result := release.VerifyCandidate(candidate, input, validation)
			require.Equal(t, domain.StatusBlocked, result.Status)
			require.Equal(t, field, result.Blockers[0].Refs[0])
		})
	}
}

func TestStrictEvidenceChangeInvalidatesCandidate(t *testing.T) {
	input := validCoreInput()
	input.Profile = release.ProfileStrictRelease
	input.StrictEvidence = validStrictEvidence()
	candidate, _ := release.CreateCandidate(input)

	input.StrictEvidence.SBOMDigest = digest("f")
	result := release.VerifyCandidate(candidate, input, validUserValidation(candidate.Digest))

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, "strict_evidence", result.Blockers[0].Refs[0])
}

func TestCandidateRejectsUnknownProfile(t *testing.T) {
	input := validCoreInput()
	input.Profile = release.Profile("enterprise")

	_, result := release.CreateCandidate(input)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, "profile", result.Blockers[0].Refs[0])
}

func validCoreInput() release.Input {
	return release.Input{
		Profile:             release.ProfileCore,
		Version:             "1.0.0",
		RootCommit:          strings.Repeat("a", 40),
		WorkspaceCommits:    map[string]string{"workspace.root": strings.Repeat("a", 40)},
		ArtifactDigests:     map[string]string{"archive": digest("a")},
		ProductFingerprint:  digest("b"),
		DocsFingerprint:     digest("c"),
		ContractFingerprint: digest("d"),
		TDDEvidence:         map[string]string{"tests": digest("e")},
		IntegrationEvidence: map[string]string{"integration": digest("f")},
		MigrationRequired:   false,
	}
}

func validStrictEvidence() *release.StrictEvidence {
	return &release.StrictEvidence{
		SBOMDigest:          digest("1"),
		ProvenanceDigest:    digest("2"),
		SignatureDigests:    map[string]string{"checksums": digest("3")},
		SupplyChainReceipts: map[string]string{"security": digest("4")},
	}
}

func validUserValidation(candidateDigest string) release.UserValidation {
	return release.UserValidation{
		SchemaVersion:   1,
		CandidateDigest: candidateDigest,
		Confirmed:       true,
		EvidenceDigest:  digest("5"),
		VerifiedAt:      time.Now().UTC().Format(time.RFC3339),
	}
}

func digest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}
