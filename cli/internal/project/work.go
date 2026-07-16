package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
	"go.yaml.in/yaml/v3"
)

// StartWorkRequest contains the normalized scope and current observable claims.
type StartWorkRequest struct {
	Root         string
	WorkID       string
	ClaimID      string
	Owner        string
	Branch       string
	ExpiresAt    time.Time
	Candidate    policy.Candidate
	ActiveClaims []policy.Claim
	Snapshot     contextpkg.Snapshot
}

// FinishWorkRequest provides the verification evidence required to close work.
type FinishWorkRequest struct {
	WorkID   string
	Evidence []string
}

// StartWork creates a reviewable claim and branch checkpoint plan after conflict preflight.
func StartWork(request StartWorkRequest) operation.Plan {
	plan := operation.Plan{ID: "start-" + request.WorkID, Root: request.Root}
	if request.Root == "" || !ValidWorkID(request.WorkID) || !validClaimID(request.ClaimID) || request.Owner == "" || request.Candidate.Repository == "" || !request.ExpiresAt.After(request.Candidate.Now) || gitx.ValidateBranch(request.Branch) != nil {
		plan.Blockers = []domain.Item{{Code: "work.request-invalid", Message: "Work and claim IDs, owner, repository, conventional branch, and a future lease are required."}}
		return plan
	}
	report := policy.CheckConflict(request.Candidate, request.ActiveClaims, request.Snapshot)
	if report.Level != policy.ConflictClear {
		plan.Blockers = append(plan.Blockers, report.Reasons...)
		if len(plan.Blockers) == 0 {
			plan.Blockers = []domain.Item{{Code: "conflict." + string(report.Level), Message: report.NextAction}}
		}
		return plan
	}

	claim := claimDocument{
		SchemaVersion: 1, ID: request.ClaimID, WorkID: request.WorkID, Owner: request.Owner,
		Branch: request.Branch, Repository: request.Candidate.Repository, Workspace: request.Candidate.Workspace,
		Paths: request.Candidate.Paths, PolicyIDs: request.Candidate.PolicyIDs, ScenarioIDs: request.Candidate.ScenarioIDs,
		ContractIDs: request.Candidate.ContractIDs, DBEntities: request.Candidate.DBEntities, MigrationSlots: request.Candidate.MigrationSlots,
		UIFlows: request.Candidate.UIFlows, DependencyMajors: request.Candidate.DependencyMajors, RootPointer: request.Candidate.RootPointer,
		StartsAt: request.Candidate.Now.UTC(), ExpiresAt: request.ExpiresAt.UTC(),
	}
	claimData, _ := yaml.Marshal(claim)
	branchData, _ := yaml.Marshal(branchDocument{SchemaVersion: 1, WorkID: request.WorkID, ClaimID: request.ClaimID, Branch: request.Branch, Baseline: "pending-context-refresh"})
	branchKey := strings.ReplaceAll(request.Branch, "/", "-")
	plan.Files = []operation.FileChange{
		{Path: filepath.ToSlash(filepath.Join(".harness", "work", "claims", request.ClaimID+".yaml")), Content: claimData, Mode: 0o644},
		{Path: filepath.ToSlash(filepath.Join(".harness", "work", "branches", branchKey+".yaml")), Content: branchData, Mode: 0o644},
	}
	fingerprint, err := operation.StateFingerprint(plan)
	if err != nil {
		plan.Blockers = []domain.Item{{Code: "work.plan-invalid", Message: err.Error()}}
		plan.Files = nil
		return plan
	}
	plan.InitialStateFingerprint = fingerprint
	return plan
}

// FinishWork verifies evidence exists before a work item may enter done state.
func FinishWork(request FinishWorkRequest) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "work.finish", OperationID: "finish-" + request.WorkID, Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "Work cannot finish without verification evidence."}
	if request.WorkID == "" {
		result.ExitCode = domain.ExitInvalid
		result.Blockers = []domain.Item{{Code: "work.id-required", Message: "Work ID is required."}}
		return result
	}
	if len(request.Evidence) == 0 {
		result.Blockers = []domain.Item{{Code: "work.evidence-required", Message: "Provide reproducible TDD, integration, or review evidence before finishing."}}
		return result
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Work verification evidence is present and the item may be completed."
	for _, evidence := range request.Evidence {
		result.Evidence = append(result.Evidence, domain.Item{Code: "work.evidence", Message: evidence, Refs: []string{request.WorkID}})
	}
	result.NextActions = []domain.Item{{Code: "work.integrate", Message: fmt.Sprintf("Integrate %s using the approved change-bundle order.", request.WorkID)}}
	return result
}

type claimDocument struct {
	SchemaVersion    int       `yaml:"schema_version"`
	ID               string    `yaml:"id"`
	WorkID           string    `yaml:"work_id"`
	Owner            string    `yaml:"owner"`
	Branch           string    `yaml:"branch"`
	Repository       string    `yaml:"repository"`
	Workspace        string    `yaml:"workspace,omitempty"`
	Paths            []string  `yaml:"paths"`
	PolicyIDs        []string  `yaml:"policy_ids"`
	ScenarioIDs      []string  `yaml:"scenario_ids"`
	ContractIDs      []string  `yaml:"contract_ids"`
	DBEntities       []string  `yaml:"db_entities"`
	MigrationSlots   []string  `yaml:"migration_slots"`
	UIFlows          []string  `yaml:"ui_flows"`
	DependencyMajors []string  `yaml:"dependency_majors"`
	RootPointer      bool      `yaml:"root_pointer"`
	StartsAt         time.Time `yaml:"starts_at"`
	ExpiresAt        time.Time `yaml:"expires_at"`
}

type branchDocument struct {
	SchemaVersion int    `yaml:"schema_version"`
	WorkID        string `yaml:"work_id"`
	ClaimID       string `yaml:"claim_id"`
	Branch        string `yaml:"branch"`
	Baseline      string `yaml:"baseline"`
}
