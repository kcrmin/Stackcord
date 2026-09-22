package evidence

import "time"

// ApprovedCommand is loaded from a reviewed workspace command manifest, never issue free text.
type ApprovedCommand struct {
	ID             string   `json:"id" yaml:"id"`
	Kind           string   `json:"kind" yaml:"kind"`
	Argv           []string `json:"argv" yaml:"argv"`
	TimeoutSeconds int      `json:"timeout_seconds" yaml:"timeout_seconds"`
	Environment    []string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// CommandManifest contains the reviewed direct commands for one workspace.
type CommandManifest struct {
	SchemaVersion int               `json:"schema_version" yaml:"schema_version"`
	WorkspaceID   string            `json:"workspace_id" yaml:"workspace_id"`
	Commands      []ApprovedCommand `json:"commands" yaml:"commands"`
}

// Request binds one approved command to exact work and contract meaning.
type Request struct {
	Workspace             string
	Repository            string
	WorkspaceID           string
	WorkID                string
	DefinitionFingerprint string
	ContractFingerprint   string
	Command               ApprovedCommand
	ArtifactDigests       map[string]string
}

// Record is reusable evidence only while every bound identity still matches.
type Record struct {
	SchemaVersion         int               `json:"schema_version" yaml:"schema_version"`
	ID                    string            `json:"id" yaml:"id"`
	Kind                  string            `json:"kind" yaml:"kind"`
	WorkID                string            `json:"work_id" yaml:"work_id"`
	WorkspaceID           string            `json:"workspace_id" yaml:"workspace_id"`
	Command               []string          `json:"command,omitempty" yaml:"command,omitempty"`
	StartedAt             time.Time         `json:"started_at" yaml:"started_at"`
	FinishedAt            time.Time         `json:"finished_at" yaml:"finished_at"`
	ExitCode              int               `json:"exit_code" yaml:"exit_code"`
	Commit                string            `json:"commit" yaml:"commit"`
	DefinitionFingerprint string            `json:"definition_fingerprint" yaml:"definition_fingerprint"`
	ContractFingerprint   string            `json:"contract_fingerprint" yaml:"contract_fingerprint"`
	OutputDigest          string            `json:"output_digest" yaml:"output_digest"`
	ArtifactDigests       map[string]string `json:"artifact_digests,omitempty" yaml:"artifact_digests,omitempty"`
}

// Actual is the current identity against which a stored record is checked.
type Actual struct {
	Workspace             string
	Repository            string
	Head                  string
	DefinitionFingerprint string
	ContractFingerprint   string
}
