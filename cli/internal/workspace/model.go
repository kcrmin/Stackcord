package workspace

import (
	"fmt"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
)

// RootSource states which authoritative relationship located the orchestration root.
type RootSource string

const (
	RootFromSuperproject RootSource = "git-superproject"
	RootFromAncestor     RootSource = "ancestor-manifest"
)

// Entry declares one code or orchestration workspace in the service boundary.
type Entry struct {
	ID                  string   `json:"id" yaml:"id"`
	Kind                string   `json:"kind" yaml:"kind"`
	Path                string   `json:"path" yaml:"path"`
	Repository          string   `json:"repository,omitempty" yaml:"repository,omitempty"`
	Remote              string   `json:"remote,omitempty" yaml:"remote,omitempty"`
	Responsibilities    []string `json:"responsibilities" yaml:"responsibilities"`
	Dependencies        []string `json:"dependencies" yaml:"dependencies"`
	ContractFingerprint string   `json:"contract_fingerprint,omitempty" yaml:"contract_fingerprint,omitempty"`
	CommandsPath        string   `json:"commands_path,omitempty" yaml:"commands_path,omitempty"`
}

// Manifest is canonical, committed service workspace topology.
type Manifest struct {
	SchemaVersion int     `json:"schema_version" yaml:"schema_version"`
	ProjectID     string  `json:"project_id" yaml:"project_id"`
	RootRemote    string  `json:"root_remote,omitempty" yaml:"root_remote,omitempty"`
	Workspaces    []Entry `json:"workspaces" yaml:"workspaces"`
}

// Bridge is the minimal service identity stored by an independently cloned child.
type Bridge struct {
	SchemaVersion       int    `json:"schema_version" yaml:"schema_version"`
	ProjectID           string `json:"project_id" yaml:"project_id"`
	RootRemote          string `json:"root_remote" yaml:"root_remote"`
	WorkspaceID         string `json:"workspace_id" yaml:"workspace_id"`
	Discovery           string `json:"discovery" yaml:"discovery"`
	ContractFingerprint string `json:"contract_fingerprint" yaml:"contract_fingerprint"`
	CommandsPath        string `json:"commands_path" yaml:"commands_path"`
}

// Root identifies the orchestration root and the workspace containing the caller.
type Root struct {
	Path               string     `json:"path"`
	CurrentWorkspaceID string     `json:"current_workspace_id"`
	Source             RootSource `json:"source"`
	Manifest           Manifest   `json:"manifest"`
}

// State combines canonical workspace identity with actual Git evidence.
type State struct {
	Entry       Entry         `json:"entry"`
	Git         gitx.State    `json:"git"`
	ExpectedSHA string        `json:"expected_sha,omitempty"`
	Confidence  string        `json:"confidence"`
	Issues      []domain.Item `json:"issues"`
}

// IncompleteContextError means a child knows its service identity but the root is unavailable.
type IncompleteContextError struct {
	ProjectID   string
	WorkspaceID string
	RootRemote  string
}

func (errorValue *IncompleteContextError) Error() string {
	return fmt.Sprintf(
		"workspace %s belongs to project %s, but orchestration root %s is not available locally",
		errorValue.WorkspaceID,
		errorValue.ProjectID,
		errorValue.RootRemote,
	)
}
