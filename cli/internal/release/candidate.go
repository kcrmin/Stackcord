package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"fullstack-orchestrator/cli/internal/domain"
)

// Input fixes every source and evidence identity included in a release candidate.
type Input struct {
	Version              string            `json:"version"`
	RootCommit           string            `json:"root_commit"`
	WorkspaceCommits     map[string]string `json:"workspace_commits"`
	ArtifactDigests      map[string]string `json:"artifact_digests"`
	SchemaVersions       map[string]string `json:"schema_versions"`
	AdapterVersions      map[string]string `json:"adapter_versions"`
	SBOMDigest           string            `json:"sbom_digest"`
	ProvenanceDigest     string            `json:"provenance_digest"`
	SignatureDigests     map[string]string `json:"signature_digests"`
	GateReceipts         map[string]string `json:"gate_receipts"`
	DocsFingerprint      string            `json:"docs_fingerprint"`
	UserValidationDigest string            `json:"user_validation_digest"`
	Gates                Gates             `json:"gates"`
}

// Candidate is immutable by digest and contains no credentials.
type Candidate struct {
	SchemaVersion int    `json:"schema_version"`
	Input         Input  `json:"input"`
	Digest        string `json:"digest"`
}

// CreateCandidate verifies gates and computes one deterministic manifest digest.
func CreateCandidate(input Input) (Candidate, domain.Result) {
	verification := Verify(input.Gates)
	if verification.Status != domain.StatusPassed {
		return Candidate{}, verification
	}
	if blockers := validateInput(input); len(blockers) > 0 {
		verification.Status, verification.ExitCode, verification.Summary = domain.StatusBlocked, domain.ExitVerification, "Release evidence identities are incomplete."
		verification.Blockers = blockers
		return Candidate{}, verification
	}
	candidate := Candidate{SchemaVersion: 1, Input: cloneInput(input)}
	candidate.Input.Gates = Gates{}
	digest, err := candidateDigest(candidate)
	if err != nil {
		verification.Status, verification.ExitCode, verification.Summary = domain.StatusFailed, domain.ExitInternal, "Release candidate could not be encoded."
		return Candidate{}, verification
	}
	candidate.Digest = digest
	verification.Command, verification.OperationID, verification.Summary = "rc.create", "rc-"+input.Version, "Immutable release candidate created from verified inputs."
	verification.Evidence = []domain.Item{{Code: "release.candidate-digest", Message: candidate.Digest}}
	return candidate, verification
}

// VerifyCandidate compares every immutable field and digest.
func VerifyCandidate(candidate Candidate, current Input) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "rc.verify", OperationID: "rc-verify", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Release candidate inputs are unchanged."}
	result.Blockers = append(result.Blockers, validateInput(candidate.Input)...)
	if candidate.SchemaVersion != 1 {
		result.Blockers = append(result.Blockers, domain.Item{Code: "release.candidate-changed", Message: "Release candidate schema version is unsupported.", Refs: []string{"schema_version"}})
	}
	expectedDigest, err := candidateDigest(candidate)
	if err != nil || candidate.Digest != expectedDigest {
		result.Blockers = append(result.Blockers, domain.Item{Code: "release.candidate-changed", Message: "Release candidate manifest digest does not match its contents.", Refs: []string{"digest"}})
	}
	fields := []struct {
		name string
		same bool
	}{
		{"version", candidate.Input.Version == current.Version},
		{"root_commit", candidate.Input.RootCommit == current.RootCommit},
		{"workspace_commits", reflect.DeepEqual(candidate.Input.WorkspaceCommits, current.WorkspaceCommits)},
		{"artifact_digests", reflect.DeepEqual(candidate.Input.ArtifactDigests, current.ArtifactDigests)},
		{"schema_versions", reflect.DeepEqual(candidate.Input.SchemaVersions, current.SchemaVersions)},
		{"adapter_versions", reflect.DeepEqual(candidate.Input.AdapterVersions, current.AdapterVersions)},
		{"sbom_digest", candidate.Input.SBOMDigest == current.SBOMDigest},
		{"provenance_digest", candidate.Input.ProvenanceDigest == current.ProvenanceDigest},
		{"signature_digests", reflect.DeepEqual(candidate.Input.SignatureDigests, current.SignatureDigests)},
		{"gate_receipts", reflect.DeepEqual(candidate.Input.GateReceipts, current.GateReceipts)},
		{"docs_fingerprint", candidate.Input.DocsFingerprint == current.DocsFingerprint},
		{"user_validation_digest", candidate.Input.UserValidationDigest == current.UserValidationDigest},
	}
	for _, field := range fields {
		if !field.same {
			result.Blockers = append(result.Blockers, domain.Item{Code: "release.candidate-changed", Message: "Release candidate input changed.", Refs: []string{field.name}})
		}
	}
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Release candidate is no longer immutable; create a new candidate."
	}
	return result
}

func validateInput(input Input) []domain.Item {
	checks := []struct {
		name string
		ok   bool
	}{
		{"version", input.Version != ""},
		{"root_commit", input.RootCommit != ""},
		{"workspace_commits", nonEmptyMap(input.WorkspaceCommits)},
		{"artifact_digests", nonEmptyMap(input.ArtifactDigests)},
		{"schema_versions", nonEmptyMap(input.SchemaVersions)},
		{"adapter_versions", nonEmptyMap(input.AdapterVersions)},
		{"sbom_digest", input.SBOMDigest != ""},
		{"provenance_digest", input.ProvenanceDigest != ""},
		{"signature_digests", nonEmptyMap(input.SignatureDigests)},
		{"gate_receipts", nonEmptyMap(input.GateReceipts)},
		{"docs_fingerprint", input.DocsFingerprint != ""},
		{"user_validation_digest", input.UserValidationDigest != ""},
	}
	var blockers []domain.Item
	for _, check := range checks {
		if !check.ok {
			blockers = append(blockers, domain.Item{Code: "release.evidence-required", Message: "Release evidence identity is required.", Refs: []string{check.name}})
		}
	}
	return blockers
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

func candidateDigest(candidate Candidate) (string, error) {
	candidate.Digest = ""
	data, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func cloneInput(input Input) Input {
	copy := input
	copy.WorkspaceCommits = cloneMap(input.WorkspaceCommits)
	copy.ArtifactDigests = cloneMap(input.ArtifactDigests)
	copy.SchemaVersions = cloneMap(input.SchemaVersions)
	copy.AdapterVersions = cloneMap(input.AdapterVersions)
	copy.SignatureDigests = cloneMap(input.SignatureDigests)
	copy.GateReceipts = cloneMap(input.GateReceipts)
	copy.Gates.Warnings = append([]Warning(nil), input.Gates.Warnings...)
	return copy
}
func cloneMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
