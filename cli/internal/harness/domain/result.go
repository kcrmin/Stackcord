package domain

// Status is the machine-readable outcome of a command.
type Status string

const (
	StatusPassed           Status = "passed"
	StatusWarning          Status = "warning"
	StatusBlocked          Status = "blocked"
	StatusFailed           Status = "failed"
	StatusUnknown          Status = "unknown"
	StatusPartial          Status = "partial"
	StatusApprovalRequired Status = "approval_required"
)

const (
	ExitSuccess          = 0
	ExitInvalid          = 2
	ExitApprovalRequired = 3
	ExitBlocked          = 4
	ExitVerification     = 5
	ExitUnavailable      = 6
	ExitPartial          = 7
	ExitInternal         = 8
)

// Approval explains whether a command needs consent before it can mutate state.
type Approval struct {
	Required bool   `json:"required"`
	Class    string `json:"class"`
	Reason   string `json:"reason"`
}

// Item is a stable, referenceable fact, warning, blocker, change, or action.
type Item struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Refs    []string `json:"refs,omitempty"`
}

// Project identifies the inspected project without embedding source content.
type Project struct {
	ID   string `json:"id,omitempty"`
	Root string `json:"root,omitempty"`
}

// Result is the stable envelope shared by people, agents, and CI.
type Result struct {
	SchemaVersion string   `json:"schema_version"`
	ToolVersion   string   `json:"tool_version"`
	Command       string   `json:"command"`
	OperationID   string   `json:"operation_id"`
	Status        Status   `json:"status"`
	ExitCode      int      `json:"exit_code"`
	Summary       string   `json:"summary"`
	Project       *Project `json:"project,omitempty"`
	Facts         []Item   `json:"facts"`
	Warnings      []Item   `json:"warnings"`
	Blockers      []Item   `json:"blockers"`
	Changes       []Item   `json:"changes"`
	Evidence      []Item   `json:"evidence"`
	NextActions   []Item   `json:"next_actions"`
	Approval      Approval `json:"approval"`
	TimingMS      int64    `json:"timing_ms"`
}

// Normalize fills collection and approval defaults so JSON output is stable.
func (r Result) Normalize() Result {
	if r.Facts == nil {
		r.Facts = []Item{}
	}
	if r.Warnings == nil {
		r.Warnings = []Item{}
	}
	if r.Blockers == nil {
		r.Blockers = []Item{}
	}
	if r.Changes == nil {
		r.Changes = []Item{}
	}
	if r.Evidence == nil {
		r.Evidence = []Item{}
	}
	if r.NextActions == nil {
		r.NextActions = []Item{}
	}
	if r.Approval.Class == "" {
		r.Approval.Class = "A"
	}
	return r
}
