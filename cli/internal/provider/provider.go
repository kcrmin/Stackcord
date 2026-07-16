package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
)

// Capability is an observable adapter feature, never an inferred provider promise.
type Capability string

const (
	CapabilityRead        Capability = "read"
	CapabilityWrite       Capability = "write"
	CapabilityHierarchy   Capability = "hierarchy"
	CapabilityDependency  Capability = "dependency"
	CapabilityDraftReview Capability = "draft_review"
	CapabilityRelease     Capability = "release"
	CapabilityDiagramSync Capability = "diagram_sync"
)

// Descriptor identifies one adapter implementation.
type Descriptor struct{ ID, Name, Version string }

// Health is explicit provider availability.
type Health struct{ Status, Message string }

// Request is a normalized provider operation.
type Request struct {
	OperationID string
	Objective   string
	Repository  string
	Target      string
	Capability  Capability
	Payload     map[string]any
}

// Discovery is provider actual state with unknowns preserved.
type Discovery struct{ Facts, Warnings, Unknown []domain.Item }

// ExecutionReceipt prevents duplicate external mutation.
type ExecutionReceipt struct {
	OperationID, Provider, Target, Fingerprint string
	CompletedAt                                time.Time
}

// Adapter is implemented by every optional external integration.
type Adapter interface {
	Descriptor() Descriptor
	Discover(context.Context, string) Discovery
	Capabilities(context.Context) []Capability
	Health(context.Context) Health
	Plan(context.Context, Request) operation.Plan
	Execute(context.Context, Request, policy.Consent) domain.Result
	Normalize([]byte) ([]domain.Item, error)
	Receipt(string) (ExecutionReceipt, bool)
}

// Executor performs one already-authorized provider write.
type Executor func(context.Context, Request) error

// GuardedConfig configures a safe first-party adapter.
type GuardedConfig struct {
	Descriptor   Descriptor
	Capabilities []Capability
	Health       Health
	Execute      Executor
}

// Guarded is a capability-negotiated, approval-checked adapter base.
type Guarded struct {
	config   GuardedConfig
	mu       sync.Mutex
	receipts map[string]ExecutionReceipt
}

// NewGuarded creates an adapter with no implicit network behavior.
func NewGuarded(config GuardedConfig) *Guarded {
	caps := append([]Capability(nil), config.Capabilities...)
	sort.Slice(caps, func(i, j int) bool { return caps[i] < caps[j] })
	config.Capabilities = caps
	return &Guarded{config: config, receipts: map[string]ExecutionReceipt{}}
}

func (adapter *Guarded) Descriptor() Descriptor { return adapter.config.Descriptor }
func (adapter *Guarded) Capabilities(context.Context) []Capability {
	return append([]Capability(nil), adapter.config.Capabilities...)
}
func (adapter *Guarded) Health(context.Context) Health { return adapter.config.Health }

func (adapter *Guarded) Discover(_ context.Context, root string) Discovery {
	if adapter.config.Health.Status != "ready" {
		return Discovery{Facts: []domain.Item{}, Warnings: []domain.Item{}, Unknown: []domain.Item{{Code: "provider.state-unknown", Message: adapter.config.Health.Message, Refs: []string{adapter.config.Descriptor.ID, root}}}}
	}
	if !HasCapability(adapter.config.Capabilities, CapabilityRead) {
		return Discovery{Facts: []domain.Item{}, Warnings: []domain.Item{{Code: "provider.capability-unsupported", Message: "Provider does not support read discovery.", Refs: []string{adapter.config.Descriptor.ID}}}, Unknown: []domain.Item{}}
	}
	return Discovery{Facts: []domain.Item{{Code: "provider.available", Message: adapter.config.Descriptor.Name}}, Warnings: []domain.Item{}, Unknown: []domain.Item{}}
}

func (adapter *Guarded) Plan(_ context.Context, request Request) operation.Plan {
	plan := operation.Plan{ID: request.OperationID, Root: request.Repository}
	if !HasCapability(adapter.config.Capabilities, request.Capability) {
		plan.Blockers = []domain.Item{{Code: "provider.capability-unsupported", Message: fmt.Sprintf("%s does not support %s", adapter.config.Descriptor.ID, request.Capability)}}
		return plan
	}
	plan.Commands = []operation.CommandStep{{Program: adapter.config.Descriptor.ID, Args: []string{string(request.Capability), request.Target}, Directory: request.Repository, ApprovalClass: approvalClass(request.Capability)}}
	return plan
}

func (adapter *Guarded) Execute(ctx context.Context, request Request, consent policy.Consent) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "provider.execute", OperationID: request.OperationID, Status: domain.StatusUnknown, ExitCode: domain.ExitUnavailable, Summary: "Provider operation is unavailable."}
	if !HasCapability(adapter.config.Capabilities, request.Capability) {
		result.Status, result.Summary = domain.StatusWarning, "Provider capability is unsupported; no state was fabricated."
		result.Warnings = []domain.Item{{Code: "provider.capability-unsupported", Message: string(request.Capability)}}
		return result
	}
	if request.Capability != CapabilityRead {
		decision := policy.Classify(policy.PushBranch, consent, policy.Scope{Objective: request.Objective, Repository: request.Repository, Target: request.Target, Now: time.Now().UTC()})
		if decision.Required {
			result.Status, result.ExitCode, result.Summary = domain.StatusApprovalRequired, domain.ExitApprovalRequired, "Exact current-scope approval is required before provider write."
			result.Approval = domain.Approval{Required: true, Class: "C", Reason: decision.Reason}
			return result
		}
	}
	adapter.mu.Lock()
	if receipt, exists := adapter.receipts[request.OperationID]; exists {
		adapter.mu.Unlock()
		result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Provider operation already completed; existing receipt was reused."
		result.Evidence = []domain.Item{{Code: "provider.receipt", Message: receipt.Fingerprint}}
		return result
	}
	adapter.mu.Unlock()
	if adapter.config.Health.Status != "ready" || adapter.config.Execute == nil {
		result.Blockers = []domain.Item{{Code: "provider.unavailable", Message: adapter.config.Health.Message}}
		return result
	}
	if err := adapter.config.Execute(ctx, request); err != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusFailed, domain.ExitInternal, "Provider operation failed without a completion receipt."
		result.Blockers = []domain.Item{{Code: "provider.execute-failed", Message: redact(err.Error())}}
		return result
	}
	receipt := ExecutionReceipt{OperationID: request.OperationID, Provider: adapter.config.Descriptor.ID, Target: request.Target, Fingerprint: requestFingerprint(request), CompletedAt: time.Now().UTC()}
	adapter.mu.Lock()
	adapter.receipts[request.OperationID] = receipt
	adapter.mu.Unlock()
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Provider operation completed and receipt was recorded."
	result.Evidence = []domain.Item{{Code: "provider.receipt", Message: receipt.Fingerprint}}
	return result
}

func (adapter *Guarded) Normalize(raw []byte) ([]domain.Item, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("provider response is malformed: %w", err)
	}
	data, _ := json.Marshal(value)
	return []domain.Item{{Code: "provider.normalized", Message: redact(string(data))}}, nil
}

func (adapter *Guarded) Receipt(id string) (ExecutionReceipt, bool) {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	receipt, ok := adapter.receipts[id]
	return receipt, ok
}

// HasCapability tests exact declared support.
func HasCapability(capabilities []Capability, capability Capability) bool {
	for _, current := range capabilities {
		if current == capability {
			return true
		}
	}
	return false
}

func approvalClass(capability Capability) string {
	if capability == CapabilityRead {
		return "A"
	}
	return "C"
}

var secretValue = regexp.MustCompile(`(?i)(token|password|secret|private[_-]?key)(=|%3D|\"\s*:\s*\")[^\"&\s]+`)

func redact(value string) string {
	return secretValue.ReplaceAllStringFunc(value, func(match string) string {
		if index := strings.IndexAny(match, "=:"); index >= 0 {
			return match[:index+1] + "[REDACTED]"
		}
		return "[REDACTED]"
	})
}

func requestFingerprint(request Request) string {
	data, _ := json.Marshal(request)
	return fmt.Sprintf("provider:%x", len(data)*131+len(request.OperationID))
}
