// Package coordination preserves project work independently of AI sessions.
// It invokes neither Git nor models. A completion is a worker report, not proof
// of code verification. Callers must fence external effects independently.
package coordination

import (
	"errors"
	"time"
)

const MaxMutationBytes = 64 << 10
const DefaultContextBytes = 16 << 10

// MaxContextBytes accommodates a task plus all 32 bounded dependency artifact sets.
const MaxContextBytes = 1 << 20

var (
	ErrInvalid  = errors.New("invalid request")
	ErrNotFound = errors.New("task not found")
	ErrConflict = errors.New("state or ownership conflict")
	ErrBlocked  = errors.New("task blocked")
	ErrBudget   = errors.New("required context exceeds byte budget")
)

type Reference struct {
	URI    string `json:"uri"`
	Digest string `json:"digest,omitempty"`
}
type Definition struct {
	Goal         string      `json:"goal"`
	Constraints  []string    `json:"constraints,omitempty"`
	Inputs       []Reference `json:"inputs,omitempty"`
	Dependencies []string    `json:"dependencies,omitempty"`
}
type Checkpoint struct {
	Summary   string      `json:"summary,omitempty"`
	Next      string      `json:"next,omitempty"`
	Blockers  []string    `json:"blockers,omitempty"`
	Artifacts []Reference `json:"artifacts,omitempty"`
}

// Usage contains only reported tokens. Nil means no observations, not zero cost.
// Each mutation carries an increment; idempotency prevents double counting.
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}
type Task struct {
	Project    string      `json:"project"`
	ID         string      `json:"id"`
	Definition Definition  `json:"definition"`
	State      string      `json:"state"`
	Owner      string      `json:"owner,omitempty"`
	Session    string      `json:"session,omitempty"`
	Epoch      uint64      `json:"epoch"`
	LeaseUntil time.Time   `json:"lease_until"`
	Revision   uint64      `json:"revision"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Checkpoint *Checkpoint `json:"checkpoint,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Usage      *Usage      `json:"reported_usage,omitempty"`
}

// Mutation.ID is unique within a project. Repeating exactly the same mutation
// returns its original result. Reusing its ID with different fields is an error.
// Requeue requires ExpectedRevision; running updates require the lease tuple.
type Mutation struct {
	ID               string      `json:"id"`
	Project          string      `json:"project"`
	Task             string      `json:"task"`
	Kind             string      `json:"kind"`
	Definition       *Definition `json:"definition,omitempty"`
	Owner            string      `json:"owner,omitempty"`
	Session          string      `json:"session,omitempty"`
	Epoch            uint64      `json:"epoch,omitempty"`
	LeaseSeconds     int         `json:"lease_seconds,omitempty"`
	ExpectedRevision uint64      `json:"expected_revision,omitempty"`
	Checkpoint       *Checkpoint `json:"checkpoint,omitempty"`
	Reason           string      `json:"reason,omitempty"`
	Usage            *Usage      `json:"usage,omitempty"`
}
type Event struct {
	Sequence uint64    `json:"sequence"`
	Task     string    `json:"task"`
	Kind     string    `json:"kind"`
	State    string    `json:"state"`
	Revision uint64    `json:"revision"`
	Owner    string    `json:"owner,omitempty"`
	Epoch    uint64    `json:"epoch"`
	At       time.Time `json:"at"`
}
type Page struct {
	Events []Event `json:"events"`
	Next   uint64  `json:"next"`
	More   bool    `json:"more"`
}
type Dependency struct {
	ID        string      `json:"id"`
	State     string      `json:"state"`
	Revision  uint64      `json:"revision"`
	Artifacts []Reference `json:"artifacts,omitempty"`
}
type Context struct {
	Task         Task         `json:"task"`
	Dependencies []Dependency `json:"dependencies"`
}
