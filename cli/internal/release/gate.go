package release

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
)

// Profile controls optional release checks without weakening core verification.
type Profile string

const (
	ProfileCore          Profile = "core"
	ProfileStrictRelease Profile = "strict-release"
)

// StrictEvidence is required only for teams that enable the strict release profile.
type StrictEvidence struct {
	SBOMDigest          string            `json:"sbom_digest"`
	ProvenanceDigest    string            `json:"provenance_digest"`
	SignatureDigests    map[string]string `json:"signature_digests"`
	SupplyChainReceipts map[string]string `json:"supply_chain_receipts"`
}

// UserValidation binds explicit user confirmation to one immutable candidate.
type UserValidation struct {
	SchemaVersion   int    `json:"schema_version"`
	CandidateDigest string `json:"candidate_digest"`
	Confirmed       bool   `json:"confirmed"`
	EvidenceDigest  string `json:"evidence_digest"`
	VerifiedAt      string `json:"verified_at"`
}

var sha256Digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var gitObjectID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

func validateInput(input Input) []domain.Item {
	checks := []struct {
		name string
		ok   bool
	}{
		{"profile", input.Profile == ProfileCore || input.Profile == ProfileStrictRelease},
		{"version", input.Version != ""},
		{"root_commit", isGitObjectID(input.RootCommit)},
		{"workspace_commits", gitObjectIDMap(input.WorkspaceCommits)},
		{"workspace_remotes", safeRemoteMap(input.WorkspaceRemotes)},
		{"provider_revisions", nonEmptyMap(input.ProviderRevisions)},
		{"tool_versions", nonEmptyMap(input.ToolVersions)},
		{"artifact_digests", digestMap(input.ArtifactDigests)},
		{"product_fingerprint", isDigest(input.ProductFingerprint)},
		{"docs_fingerprint", isDigest(input.DocsFingerprint)},
		{"contract_fingerprint", isDigest(input.ContractFingerprint)},
		{"tdd_evidence", digestMap(input.TDDEvidence)},
		{"integration_evidence", digestMap(input.IntegrationEvidence)},
	}
	var blockers []domain.Item
	for _, check := range checks {
		if !check.ok {
			blockers = append(blockers, required(check.name))
		}
	}
	if input.MigrationRequired {
		if !isDigest(input.MigrationEvidence) {
			blockers = append(blockers, required("migration_evidence"))
		}
		if !isDigest(input.RollbackEvidence) {
			blockers = append(blockers, required("rollback_evidence"))
		}
	}
	if input.Profile == ProfileStrictRelease {
		if input.StrictEvidence == nil {
			blockers = append(blockers, required("strict_evidence"))
			return blockers
		}
		strictChecks := []struct {
			name string
			ok   bool
		}{
			{"sbom_digest", isDigest(input.StrictEvidence.SBOMDigest)},
			{"provenance_digest", isDigest(input.StrictEvidence.ProvenanceDigest)},
			{"signature_digests", digestMap(input.StrictEvidence.SignatureDigests)},
			{"supply_chain_receipts", digestMap(input.StrictEvidence.SupplyChainReceipts)},
		}
		for _, check := range strictChecks {
			if !check.ok {
				blockers = append(blockers, required(check.name))
			}
		}
	}
	return blockers
}

func safeRemoteMap(values map[string]string) bool {
	if !nonEmptyMap(values) {
		return false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.ContainsAny(value, "\x00\r\n") || strings.HasPrefix(value, "-") || filepath.IsAbs(value) || strings.HasPrefix(strings.ToLower(value), "file:") {
			return false
		}
		parsed, err := url.Parse(value)
		if err != nil {
			return false
		}
		if parsed.Scheme != "" {
			if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
				return false
			}
			if parsed.User != nil {
				if _, hasPassword := parsed.User.Password(); hasPassword {
					return false
				}
			}
		}
	}
	return true
}

func validateUserValidation(validation UserValidation, candidateDigest string) []domain.Item {
	checks := []struct {
		name string
		ok   bool
	}{
		{"candidate_digest", validation.SchemaVersion == 1 && validation.CandidateDigest == candidateDigest && isDigest(validation.CandidateDigest)},
		{"confirmed", validation.Confirmed},
		{"evidence_digest", isDigest(validation.EvidenceDigest)},
		{"verified_at", validTime(validation.VerifiedAt)},
	}
	var blockers []domain.Item
	for _, check := range checks {
		if !check.ok {
			blockers = append(blockers, domain.Item{Code: "release.user-validation-invalid", Message: "User validation must reference the exact release candidate.", Refs: []string{check.name}})
		}
	}
	return blockers
}

func required(name string) domain.Item {
	return domain.Item{Code: "release.evidence-required", Message: "Release evidence identity is required.", Refs: []string{name}}
}

func isDigest(value string) bool {
	return sha256Digest.MatchString(value)
}

func digestMap(values map[string]string) bool {
	if len(values) == 0 {
		return false
	}
	for key, value := range values {
		if key == "" || !isDigest(value) {
			return false
		}
	}
	return true
}

func isGitObjectID(value string) bool {
	return gitObjectID.MatchString(value)
}

func gitObjectIDMap(values map[string]string) bool {
	if len(values) == 0 {
		return false
	}
	for key, value := range values {
		if key == "" || !isGitObjectID(value) {
			return false
		}
	}
	return true
}

func nonEmptyMap(values map[string]string) bool {
	if len(values) == 0 {
		return false
	}
	for key, value := range values {
		if key == "" || value == "" {
			return false
		}
	}
	return true
}

func validTime(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
