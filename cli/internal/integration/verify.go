package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

var integrationDigest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var integrationGitObject = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// Verify confirms every planned boundary against exact current Git and evidence identities.
func Verify(plan MergePlan, records []Evidence, current []WorkspaceState, contractFingerprint string) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "integrate.verify", OperationID: "integration-verify", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Every service integration boundary matches exact current evidence."}
	result.Blockers = append(result.Blockers, validateMergePlan(plan)...)
	result.Blockers = append(result.Blockers, plan.Blockers...)
	currentByID := map[string]WorkspaceState{}
	for _, state := range current {
		currentByID[state.ID] = state
		expected, exists := plan.WorkspaceCommits[state.ID]
		if !exists || expected != state.Commit || !state.Clean || !state.Published || (state.Kind == "submodule" && (state.ExpectedPointer != state.ActualPointer || state.Commit != state.ActualPointer)) {
			result.Blockers = append(result.Blockers, integrationItem("integrate.workspace-changed", "Workspace identity changed after integration planning.", state.ID))
		}
	}
	if !integrationDigest.MatchString(contractFingerprint) {
		result.Blockers = append(result.Blockers, integrationItem("integrate.contract-unknown", "Current contract fingerprint is required."))
	} else if plan.ContractFingerprint != "" && plan.ContractFingerprint != contractFingerprint {
		result.Blockers = append(result.Blockers, integrationItem("integrate.contract-changed", "Contract identity changed after integration planning."))
	}
	byStep := map[string]Evidence{}
	for _, record := range records {
		if _, duplicate := byStep[record.StepID]; duplicate {
			result.Blockers = append(result.Blockers, integrationItem("integrate.evidence-duplicate", "Integration evidence is duplicated.", record.StepID))
			continue
		}
		byStep[record.StepID] = record
	}
	verified := []Evidence{}
	for _, step := range plan.Steps {
		record, exists := byStep[step.ID]
		state, workspaceExists := currentByID[step.WorkspaceID]
		if !exists {
			result.Blockers = append(result.Blockers, integrationItem("integrate.evidence-missing", "Integration step has no exact evidence.", step.ID))
			continue
		}
		if !workspaceExists || record.StepID != step.ID || record.WorkID != step.WorkID || record.Kind != step.RequiredEvidence || record.WorkspaceID != step.WorkspaceID ||
			record.DefinitionFingerprint != step.DefinitionFingerprint || record.ContractFingerprint != contractFingerprint || record.ProviderRevision != step.ProviderRevision ||
			record.Commit != step.Commit || record.Commit != state.Commit || !integrationDigest.MatchString(record.Digest) {
			result.Blockers = append(result.Blockers, integrationItem("integrate.evidence-stale", "Integration evidence differs from the planned product, provider, workspace, contract, or commit identity.", step.ID))
			continue
		}
		verified = append(verified, record)
	}
	result.Blockers = normalizeIntegrationItems(result.Blockers)
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Service integration is incomplete or no longer matches current state."
		return result
	}
	data, _ := json.Marshal(verified)
	digest := sha256.Sum256(data)
	value := "sha256:" + hex.EncodeToString(digest[:])
	result.OperationID = "integration-verify-" + hex.EncodeToString(digest[:6])
	result.Evidence = []domain.Item{{Code: "integration.digest", Message: value}}
	return result
}

func validateMergePlan(plan MergePlan) []domain.Item {
	issues := []domain.Item{}
	if plan.SchemaVersion != 1 || len(plan.Steps) == 0 || len(plan.WorkspaceCommits) == 0 {
		issues = append(issues, integrationItem("integrate.plan-invalid", "Integration plan schema, steps, or workspace identities are incomplete."))
	}
	if plan.ContractFingerprint != "" && !integrationDigest.MatchString(plan.ContractFingerprint) {
		issues = append(issues, integrationItem("integrate.plan-invalid", "Integration plan contract identity is invalid."))
	}
	for id, commit := range plan.WorkspaceCommits {
		if id == "" || !integrationGitObject.MatchString(commit) {
			issues = append(issues, integrationItem("integrate.plan-invalid", "Integration plan workspace commit is invalid.", id))
		}
	}
	seen := map[string]bool{}
	for _, step := range plan.Steps {
		if step.ID == "" || seen[step.ID] || step.WorkID == "" || step.Ref == "" || step.WorkspaceID == "" || !integrationDigest.MatchString(step.DefinitionFingerprint) || step.ProviderRevision == "" || !integrationGitObject.MatchString(step.Commit) || plan.WorkspaceCommits[step.WorkspaceID] != step.Commit || !validIntegrationEvidenceKind(step.RequiredEvidence) {
			issues = append(issues, integrationItem("integrate.plan-invalid", "Integration step identity is missing, duplicated, or inconsistent.", step.ID))
		}
		seen[step.ID] = true
	}
	for _, step := range plan.Steps {
		for _, dependency := range step.DependsOn {
			if !seen[dependency] || dependency == step.ID {
				issues = append(issues, integrationItem("integrate.plan-invalid", "Integration step dependency is missing or self-referential.", step.ID, dependency))
			}
		}
	}
	return normalizeIntegrationItems(issues)
}

func validIntegrationEvidenceKind(value string) bool {
	switch value {
	case "review", "integration", "child-merge", "migration", "root-pointer":
		return true
	default:
		return false
	}
}
