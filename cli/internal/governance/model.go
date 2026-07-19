package governance

import (
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

// Status is the effective product-meaning approval state.
type Status string

const (
	Disabled Status = "disabled"
	Proposed Status = "proposed"
	Approved Status = "approved"
	Unknown  Status = "unknown"
	Blocked  Status = "blocked"
)

// ApprovalPolicy controls how many configured authorities must approve.
type ApprovalPolicy struct {
	Minimum               int  `json:"minimum" yaml:"minimum"`
	AuthoritySelfApproval bool `json:"authority_self_approval" yaml:"authority_self_approval"`
}

// Policy is committed product-governance configuration.
type Policy struct {
	SchemaVersion      int            `json:"schema_version" yaml:"schema_version"`
	Enabled            bool           `json:"enabled" yaml:"enabled"`
	Provider           string         `json:"provider" yaml:"provider"`
	Repository         string         `json:"repository" yaml:"repository"`
	ProductAuthorities []string       `json:"product_authorities" yaml:"product_authorities"`
	ProtectedKinds     []string       `json:"protected_kinds" yaml:"protected_kinds"`
	Approval           ApprovalPolicy `json:"approval" yaml:"approval"`
}

// Decision is one normalized account action observed by a selected Git provider.
type Decision struct {
	Subject     string    `json:"subject" yaml:"subject"`
	Kind        string    `json:"kind" yaml:"kind"`
	State       string    `json:"state" yaml:"state"`
	Revision    string    `json:"revision" yaml:"revision"`
	SubmittedAt time.Time `json:"submitted_at" yaml:"submitted_at"`
}

// Observation is ignored local evidence produced from a live provider review.
type Observation struct {
	SchemaVersion        int        `json:"schema_version" yaml:"schema_version"`
	Provider             string     `json:"provider" yaml:"provider"`
	Repository           string     `json:"repository" yaml:"repository"`
	ReviewID             string     `json:"review_id" yaml:"review_id"`
	ReviewRevision       string     `json:"review_revision" yaml:"review_revision"`
	HeadCommit           string     `json:"head_commit" yaml:"head_commit"`
	ProtectedFingerprint string     `json:"protected_fingerprint" yaml:"protected_fingerprint"`
	AuthorSubject        string     `json:"author_subject" yaml:"author_subject"`
	Status               string     `json:"status" yaml:"status"`
	Decisions            []Decision `json:"decisions" yaml:"decisions"`
	FetchedAt            time.Time  `json:"fetched_at" yaml:"fetched_at"`
	Source               string     `json:"source" yaml:"source"`
	RawHash              string     `json:"raw_hash" yaml:"raw_hash"`
}

// Report contains no provider payload or credential and is safe for status output.
type Report struct {
	Enabled              bool          `json:"enabled"`
	Status               Status        `json:"status"`
	ProtectedFingerprint string        `json:"protected_fingerprint,omitempty"`
	ApprovalRevision     string        `json:"approval_revision,omitempty"`
	Authorities          []string      `json:"authorities"`
	Approvers            []string      `json:"approvers"`
	Issues               []domain.Item `json:"issues"`
}
