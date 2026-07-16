package contract

import "fullstack-orchestrator/cli/internal/domain"

// Check ensures a contract has stable identity and explicit behavioral obligations.
func Check(definition Definition) []domain.Item {
	var issues []domain.Item
	if definition.ID == "" {
		issues = append(issues, domain.Item{Code: "contract.id-required", Message: "Contract stable ID is required."})
	}
	if definition.Retry == "" || definition.Idempotency == "" || definition.Timeout == "" || definition.PartialFailure == "" || definition.Compensation == "" {
		issues = append(issues, domain.Item{Code: "contract.behavior-required", Message: "Retry, idempotency, timeout, partial-failure, and compensation obligations must be explicit."})
	}
	return issues
}
