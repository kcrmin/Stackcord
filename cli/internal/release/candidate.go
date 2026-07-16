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
	candidate := Candidate{SchemaVersion: 1, Input: cloneInput(input)}
	candidate.Input.Gates = Gates{}
	data, err := json.Marshal(candidate)
	if err != nil {
		verification.Status, verification.ExitCode, verification.Summary = domain.StatusFailed, domain.ExitInternal, "Release candidate could not be encoded."
		return Candidate{}, verification
	}
	digest := sha256.Sum256(data)
	candidate.Digest = "sha256:" + hex.EncodeToString(digest[:])
	verification.Command, verification.OperationID, verification.Summary = "rc.create", "rc-"+input.Version, "Immutable release candidate created from verified inputs."
	verification.Evidence = []domain.Item{{Code: "release.candidate-digest", Message: candidate.Digest}}
	return candidate, verification
}

// VerifyCandidate compares every immutable field and digest.
func VerifyCandidate(candidate Candidate, current Input) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "rc.verify", OperationID: "rc-verify", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Release candidate inputs are unchanged."}
	fields := []struct {
		name string
		same bool
	}{
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
