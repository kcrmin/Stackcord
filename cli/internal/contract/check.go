package contract

import (
	"regexp"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
)

var contractIDPattern = regexp.MustCompile(`^contract\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

// Check ensures a contract has stable identity and explicit behavioral obligations.
func Check(definition Definition) []domain.Item {
	var issues []domain.Item
	if !contractIDPattern.MatchString(definition.ID) {
		issues = append(issues, domain.Item{Code: "contract.id-required", Message: "Contract stable ID is required."})
	}
	kind := effectiveKind(definition.Kind)
	if kind != Product && kind != Business && kind != Behavior && kind != Interface && kind != Data {
		issues = append(issues, domain.Item{Code: "contract.kind-invalid", Message: "Contract kind must be product, business, behavior, interface, or data."})
	}
	if kind != Interface && strings.TrimSpace(definition.Purpose) == "" {
		issues = append(issues, domain.Item{Code: "contract.purpose-required", Message: "The obligation purpose must be explicit."})
	}
	switch kind {
	case Product:
		if len(definition.Rules) == 0 {
			issues = append(issues, domain.Item{Code: "contract.commitment-required", Message: "Product contracts require explicit service commitments."})
		}
		if len(definition.NonGoals) == 0 {
			issues = append(issues, domain.Item{Code: "contract.non-goal-required", Message: "Product contracts require explicit non-goals."})
		}
	case Business:
		if len(definition.Rules) == 0 || len(definition.Eligibility) == 0 || len(definition.Invariants) == 0 || len(definition.Outcomes) == 0 {
			issues = append(issues, domain.Item{Code: "contract.business-rule-required", Message: "Business contracts require rules, eligibility, invariants, and observable outcomes."})
		}
		issues = appendObservableFailures(issues, definition)
	case Behavior:
		if len(definition.Outcomes) == 0 {
			issues = append(issues, domain.Item{Code: "contract.outcome-required", Message: "Behavior contracts require observable outcomes."})
		}
		issues = appendObservableFailures(issues, definition)
		issues = appendOperationalBehavior(issues, definition)
	case Interface:
		issues = appendOperationalBehavior(issues, definition)
	case Data:
		if len(definition.Invariants) == 0 || definition.DataOwner == "" || definition.Classification == "" || definition.Retention == "" || definition.Deletion == "" || definition.Migration == "" {
			issues = append(issues, domain.Item{Code: "contract.data-lifecycle-required", Message: "Data contracts require invariants, owner, classification, retention, deletion, and migration behavior."})
		}
	}
	sort.Slice(issues, func(left, right int) bool { return issues[left].Code < issues[right].Code })
	return issues
}

func appendObservableFailures(issues []domain.Item, definition Definition) []domain.Item {
	if len(definition.Rejections) == 0 {
		issues = append(issues, domain.Item{Code: "contract.rejection-behavior-required", Message: "Observable rejected behavior must be explicit."})
	}
	if len(definition.Failures) == 0 {
		issues = append(issues, domain.Item{Code: "contract.failure-behavior-required", Message: "Observable failure behavior must be explicit."})
	}
	return issues
}

func appendOperationalBehavior(issues []domain.Item, definition Definition) []domain.Item {
	if definition.Retry == "" || definition.Idempotency == "" || definition.Timeout == "" || definition.PartialFailure == "" || definition.Compensation == "" {
		issues = append(issues, domain.Item{Code: "contract.behavior-required", Message: "Retry, idempotency, timeout, partial-failure, and compensation obligations must be explicit."})
	}
	return issues
}
