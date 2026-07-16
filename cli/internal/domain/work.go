package domain

import "time"

// WorkStatus is the normalized status shared by local and external task providers.
type WorkStatus string

const (
	WorkProposed   WorkStatus = "proposed"
	WorkReady      WorkStatus = "ready"
	WorkInProgress WorkStatus = "in_progress"
	WorkBlocked    WorkStatus = "blocked"
	WorkReview     WorkStatus = "review"
	WorkDone       WorkStatus = "done"
)

// WorkItem links execution state to canonical product meaning without duplicating it.
type WorkItem struct {
	SchemaVersion int        `json:"schema_version" yaml:"schema_version"`
	ID            string     `json:"id" yaml:"id"`
	Title         string     `json:"title" yaml:"title"`
	Status        WorkStatus `json:"status" yaml:"status"`
	Owner         string     `json:"owner,omitempty" yaml:"owner,omitempty"`
	Refs          []string   `json:"refs" yaml:"refs"`
	Dependencies  []string   `json:"dependencies" yaml:"dependencies"`
	UpdatedAt     time.Time  `json:"updated_at" yaml:"updated_at"`
}
