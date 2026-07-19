package integration

import "github.com/kcrmin/Stackcord/cli/internal/domain"

// StepKind names one service integration boundary in dependency order.
type StepKind string

const (
	ContractStep     StepKind = "contract"
	WorkspaceStep    StepKind = "workspace"
	UIConnectionStep StepKind = "ui-connection"
	MigrationStep    StepKind = "migration"
	RootPointerStep  StepKind = "root-pointer"
)

// ProviderState is one live, revisioned task-provider observation.
type ProviderState struct {
	WorkID                string `json:"work_id"`
	Status                string `json:"status"`
	Revision              string `json:"revision"`
	DefinitionFingerprint string `json:"definition_fingerprint"`
	Confirmed             bool   `json:"confirmed"`
}

// WorkspaceState binds one declared workspace to actual recoverable Git identity.
type WorkspaceState struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Commit          string `json:"commit"`
	Remote          string `json:"remote"`
	Clean           bool   `json:"clean"`
	Published       bool   `json:"published"`
	ExpectedPointer string `json:"expected_pointer,omitempty"`
	ActualPointer   string `json:"actual_pointer,omitempty"`
}

// Step binds merge order to exact product, provider, workspace, and evidence identity.
type Step struct {
	ID                    string   `json:"id"`
	Kind                  StepKind `json:"kind"`
	Ref                   string   `json:"ref"`
	WorkID                string   `json:"work_id"`
	WorkspaceID           string   `json:"workspace_id"`
	DefinitionFingerprint string   `json:"definition_fingerprint"`
	ProviderRevision      string   `json:"provider_revision"`
	Commit                string   `json:"commit"`
	RequiredEvidence      string   `json:"required_evidence"`
	DependsOn             []string `json:"depends_on"`
}

// MergePlan is deterministic and immutable once evidence is recorded against its identities.
type MergePlan struct {
	SchemaVersion              int               `json:"schema_version"`
	Steps                      []Step            `json:"steps"`
	WorkspaceCommits           map[string]string `json:"workspace_commits"`
	ContractFingerprint        string            `json:"contract_fingerprint,omitempty"`
	GovernanceFingerprint      string            `json:"governance_fingerprint,omitempty"`
	GovernanceApprovalRevision string            `json:"governance_approval_revision,omitempty"`
	Blockers                   []domain.Item     `json:"blockers"`
}

// Evidence proves one exact integration step without copying command output.
type Evidence struct {
	StepID                string `json:"step_id"`
	WorkID                string `json:"work_id"`
	Kind                  string `json:"kind"`
	WorkspaceID           string `json:"workspace_id"`
	DefinitionFingerprint string `json:"definition_fingerprint"`
	ContractFingerprint   string `json:"contract_fingerprint"`
	ProviderRevision      string `json:"provider_revision"`
	Commit                string `json:"commit"`
	Digest                string `json:"digest"`
}
