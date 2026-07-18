package continuity

import (
	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/workspace"
)

// Confidence distinguishes evidence quality instead of collapsing every concern into pass/fail.
type Confidence string

const (
	Confirmed Confidence = "confirmed"
	Warning   Confidence = "warning"
	Stale     Confidence = "stale"
	Unknown   Confidence = "unknown"
	LocalOnly Confidence = "local-only"
	Blocked   Confidence = "blocked"
)

// ProviderView is a small, secret-free view of the selected live task source.
type ProviderView struct {
	Name       string     `json:"name"`
	ItemID     string     `json:"item_id,omitempty"`
	Revision   string     `json:"revision,omitempty"`
	Owner      string     `json:"owner,omitempty"`
	Status     string     `json:"status,omitempty"`
	Confidence Confidence `json:"confidence"`
}

// WorkView identifies active work without copying task prose into status packets.
type WorkView struct {
	ID                    string `json:"id"`
	Title                 string `json:"title,omitempty"`
	State                 string `json:"state,omitempty"`
	Owner                 string `json:"owner,omitempty"`
	Branch                string `json:"branch,omitempty"`
	LiveRevision          string `json:"live_revision,omitempty"`
	DefinitionFingerprint string `json:"definition_fingerprint"`
}

// ReleaseView identifies the currently visible candidate without treating it as verified.
type ReleaseView struct {
	CandidateDigest string     `json:"candidate_digest,omitempty"`
	Confidence      Confidence `json:"confidence"`
}

// Snapshot is the deterministic, read-only continuity view consumed by people, Skills, and hooks.
type Snapshot struct {
	SchemaVersion        int                 `json:"schema_version"`
	ProjectID            string              `json:"project_id,omitempty"`
	Root                 string              `json:"root,omitempty"`
	CurrentWorkspaceID   string              `json:"current_workspace_id,omitempty"`
	CanonicalFingerprint string              `json:"canonical_fingerprint,omitempty"`
	Overall              Confidence          `json:"overall"`
	Context              contextpkg.Snapshot `json:"context"`
	Workspaces           []workspace.State   `json:"workspaces"`
	Provider             ProviderView        `json:"provider"`
	ActiveWork           []WorkView          `json:"active_work"`
	Release              ReleaseView         `json:"release"`
	Issues               []domain.Item       `json:"issues"`
	NextActions          []domain.Item       `json:"next_actions"`
}

// Options reserves deterministic collection inputs without exposing process-local caches as truth.
type Options struct{}
