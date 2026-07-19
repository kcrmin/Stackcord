package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

// Input fixes every core identity and any explicitly enabled strict identity.
type Input struct {
	Profile                    Profile           `json:"profile"`
	Version                    string            `json:"version"`
	RootCommit                 string            `json:"root_commit"`
	WorkspaceCommits           map[string]string `json:"workspace_commits"`
	WorkspaceRemotes           map[string]string `json:"workspace_remotes"`
	ProviderRevisions          map[string]string `json:"provider_revisions"`
	ToolVersions               map[string]string `json:"tool_versions"`
	ArtifactDigests            map[string]string `json:"artifact_digests"`
	ProductFingerprint         string            `json:"product_fingerprint"`
	DocsFingerprint            string            `json:"docs_fingerprint"`
	ContractFingerprint        string            `json:"contract_fingerprint"`
	GovernanceFingerprint      string            `json:"governance_fingerprint"`
	GovernanceApprovalRevision string            `json:"governance_approval_revision"`
	TDDEvidence                map[string]string `json:"tdd_evidence"`
	IntegrationEvidence        map[string]string `json:"integration_evidence"`
	MigrationRequired          bool              `json:"migration_required"`
	MigrationEvidence          string            `json:"migration_evidence,omitempty"`
	RollbackEvidence           string            `json:"rollback_evidence,omitempty"`
	StrictEvidence             *StrictEvidence   `json:"strict_evidence,omitempty"`
}

// Candidate is immutable by digest and contains no credentials or user approval.
type Candidate struct {
	SchemaVersion int    `json:"schema_version"`
	Input         Input  `json:"input"`
	Digest        string `json:"digest"`
}

// CreateCandidate validates the selected profile and computes one deterministic digest.
func CreateCandidate(input Input) (Candidate, domain.Result) {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "release.prepare", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Release candidate created from exact verified inputs."}
	if blockers := validateInput(input); len(blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Release candidate evidence is incomplete."
		result.Blockers = blockers
		return Candidate{}, result
	}
	candidate := Candidate{SchemaVersion: 1, Input: cloneInput(input)}
	digest, err := candidateDigest(candidate)
	if err != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusFailed, domain.ExitInternal, "Release candidate could not be encoded."
		return Candidate{}, result
	}
	candidate.Digest = digest
	result.OperationID = operationID("release-prepare", input.Version, candidate.Digest)
	result.Evidence = []domain.Item{{Code: "release.candidate-digest", Message: candidate.Digest}}
	return candidate, result
}

// VerifyCandidate verifies current technical identities and user confirmation against one digest.
func VerifyCandidate(candidate Candidate, current Input, validation UserValidation) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "release.verify", OperationID: operationID("release-verify", candidate.Input.Version, candidate.Digest), Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Technical and user validation reference the same release candidate."}
	result.Blockers = append(result.Blockers, ValidateCandidate(candidate)...)
	fields := []struct {
		name string
		same bool
	}{
		{"version", candidate.Input.Version == current.Version},
		{"root_commit", candidate.Input.RootCommit == current.RootCommit},
		{"workspace_commits", reflect.DeepEqual(candidate.Input.WorkspaceCommits, current.WorkspaceCommits)},
		{"workspace_remotes", reflect.DeepEqual(candidate.Input.WorkspaceRemotes, current.WorkspaceRemotes)},
		{"provider_revisions", reflect.DeepEqual(candidate.Input.ProviderRevisions, current.ProviderRevisions)},
		{"tool_versions", reflect.DeepEqual(candidate.Input.ToolVersions, current.ToolVersions)},
		{"artifact_digests", reflect.DeepEqual(candidate.Input.ArtifactDigests, current.ArtifactDigests)},
		{"product_fingerprint", candidate.Input.ProductFingerprint == current.ProductFingerprint},
		{"docs_fingerprint", candidate.Input.DocsFingerprint == current.DocsFingerprint},
		{"contract_fingerprint", candidate.Input.ContractFingerprint == current.ContractFingerprint},
		{"governance_fingerprint", candidate.Input.GovernanceFingerprint == current.GovernanceFingerprint},
		{"governance_approval_revision", candidate.Input.GovernanceApprovalRevision == current.GovernanceApprovalRevision},
		{"tdd_evidence", reflect.DeepEqual(candidate.Input.TDDEvidence, current.TDDEvidence)},
		{"integration_evidence", reflect.DeepEqual(candidate.Input.IntegrationEvidence, current.IntegrationEvidence)},
		{"migration_required", candidate.Input.MigrationRequired == current.MigrationRequired},
		{"migration_evidence", candidate.Input.MigrationEvidence == current.MigrationEvidence},
		{"rollback_evidence", candidate.Input.RollbackEvidence == current.RollbackEvidence},
		{"strict_evidence", reflect.DeepEqual(candidate.Input.StrictEvidence, current.StrictEvidence)},
	}
	for _, field := range fields {
		if !field.same {
			result.Blockers = append(result.Blockers, changed(field.name))
		}
	}
	result.Blockers = append(result.Blockers, validateInput(current)...)
	result.Blockers = append(result.Blockers, validateUserValidation(validation, candidate.Digest)...)
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Release candidate identity or validation differs; prepare or validate the exact candidate."
		return result
	}
	result.Evidence = []domain.Item{{Code: "release.candidate-digest", Message: candidate.Digest}, {Code: "release.user-validation", Message: validation.EvidenceDigest}}
	return result
}

// ValidateCandidate rejects incomplete or tampered candidate identity before user confirmation.
func ValidateCandidate(candidate Candidate) []domain.Item {
	blockers := validateInput(candidate.Input)
	if candidate.SchemaVersion != 1 {
		blockers = append(blockers, changed("schema_version"))
	}
	expectedDigest, err := candidateDigest(candidate)
	if err != nil || candidate.Digest != expectedDigest {
		blockers = append(blockers, changed("digest"))
	}
	return blockers
}

func changed(field string) domain.Item {
	return domain.Item{Code: "release.candidate-changed", Message: "Release candidate input changed.", Refs: []string{field}}
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
	copy.WorkspaceRemotes = cloneMap(input.WorkspaceRemotes)
	copy.ProviderRevisions = cloneMap(input.ProviderRevisions)
	copy.ToolVersions = cloneMap(input.ToolVersions)
	copy.ArtifactDigests = cloneMap(input.ArtifactDigests)
	copy.TDDEvidence = cloneMap(input.TDDEvidence)
	copy.IntegrationEvidence = cloneMap(input.IntegrationEvidence)
	if input.StrictEvidence != nil {
		strict := *input.StrictEvidence
		strict.SignatureDigests = cloneMap(input.StrictEvidence.SignatureDigests)
		strict.SupplyChainReceipts = cloneMap(input.StrictEvidence.SupplyChainReceipts)
		copy.StrictEvidence = &strict
	}
	return copy
}

func cloneMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func operationID(prefix, version, digest string) string {
	short := strings.TrimPrefix(digest, "sha256:")
	if len(short) > 12 {
		short = short[:12]
	}
	return prefix + "-" + version + "-" + short
}
