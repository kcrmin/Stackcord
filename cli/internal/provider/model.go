package provider

import (
	"time"

	"fullstack-orchestrator/cli/internal/domain"
)

// Confidence states whether provider facts were observed live or are unusable for coordination.
type Confidence string

const (
	Confirmed Confidence = "confirmed"
	Stale     Confidence = "stale"
	Unknown   Confidence = "unknown"
)

// Capabilities makes provider coordination limits explicit.
type Capabilities struct {
	Hierarchy    bool   `json:"hierarchy" yaml:"hierarchy"`
	Dependencies bool   `json:"dependencies" yaml:"dependencies"`
	Claim        string `json:"claim" yaml:"claim"`
	Revision     bool   `json:"revision" yaml:"revision"`
}

// Expectation is derived from canonical work without importing the work package.
type Expectation struct {
	WorkID                string   `json:"work_id"`
	DefinitionFingerprint string   `json:"definition_fingerprint"`
	Dependencies          []string `json:"dependencies"`
}

// Mapping is the only provider relationship committed to the repository.
type Mapping struct {
	SchemaVersion         int               `json:"schema_version" yaml:"schema_version"`
	WorkID                string            `json:"work_id" yaml:"work_id"`
	DefinitionFingerprint string            `json:"definition_fingerprint" yaml:"definition_fingerprint"`
	Provider              string            `json:"provider" yaml:"provider"`
	ItemID                string            `json:"item_id" yaml:"item_id"`
	DependencyItems       map[string]string `json:"dependency_items" yaml:"dependency_items"`
}

// Snapshot is normalized connector output. It is local evidence, never canonical state.
type Snapshot struct {
	SchemaVersion         int          `json:"schema_version" yaml:"schema_version"`
	Provider              string       `json:"provider" yaml:"provider"`
	ItemID                string       `json:"item_id" yaml:"item_id"`
	Revision              string       `json:"revision" yaml:"revision"`
	Status                string       `json:"status" yaml:"status"`
	Owner                 string       `json:"owner,omitempty" yaml:"owner,omitempty"`
	Dependencies          []string     `json:"dependencies" yaml:"dependencies"`
	Capabilities          Capabilities `json:"capabilities" yaml:"capabilities"`
	DefinitionFingerprint string       `json:"definition_fingerprint" yaml:"definition_fingerprint"`
	FetchedAt             time.Time    `json:"fetched_at" yaml:"fetched_at"`
	Source                string       `json:"source" yaml:"source"`
	RawHash               string       `json:"raw_hash" yaml:"raw_hash"`
}

// State is the reconciled provider truth used for claims and lifecycle decisions.
type State struct {
	Confidence Confidence    `json:"confidence"`
	Provider   string        `json:"provider"`
	ItemID     string        `json:"item_id"`
	Revision   string        `json:"revision,omitempty"`
	Status     string        `json:"status,omitempty"`
	Owner      string        `json:"owner,omitempty"`
	Issues     []domain.Item `json:"issues"`
}
