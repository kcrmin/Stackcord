package integration

import (
	"github.com/kcrmin/Stackcord/cli/internal/contract"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/work"
)

// CheckCompatibility validates contract approval and provider-before-consumer ordering for release work.
func CheckCompatibility(definitions []work.Definition, registry contract.Registry) []domain.Item {
	entries := map[string]contract.Entry{}
	for _, entry := range registry.Contracts {
		entries[entry.ID] = entry
	}
	issues := []domain.Item{}
	for _, definition := range definitions {
		order := map[string]int{}
		for index, id := range definition.MergeOrder {
			order[id] = index
		}
		for _, contractID := range definition.Scope.ContractIDs {
			entry, exists := entries[contractID]
			if !exists {
				issues = append(issues, integrationItem("integrate.contract-missing", "Work references an unregistered contract.", definition.ID, contractID))
				continue
			}
			if entry.Status != contract.Approved {
				issues = append(issues, integrationItem("integrate.contract-unapproved", "Integration requires an approved current contract.", definition.ID, contractID, string(entry.Status)))
			}
			if entry.Compatibility == contract.Additive {
				continue
			}
			if len(entry.Providers) == 0 || len(entry.Consumers) == 0 {
				issues = append(issues, integrationItem("integrate.compatibility-participants", "Coordinated or breaking contracts require explicit providers and consumers.", definition.ID, contractID))
				continue
			}
			maxProvider, minConsumer := -1, len(definition.MergeOrder)+1
			missing := false
			for _, providerID := range entry.Providers {
				index, found := order[providerID]
				if !found {
					missing = true
					continue
				}
				if index > maxProvider {
					maxProvider = index
				}
			}
			for _, consumerID := range entry.Consumers {
				index, found := order[consumerID]
				if !found {
					missing = true
					continue
				}
				if index < minConsumer {
					minConsumer = index
				}
			}
			if missing || maxProvider >= minConsumer {
				issues = append(issues, integrationItem("integrate.compatibility-order", "Every provider must precede every consumer in merge order.", definition.ID, contractID))
			}
			if entry.Compatibility == contract.Breaking && !definition.Evidence.IntegrationRequired {
				issues = append(issues, integrationItem("integrate.breaking-evidence", "Breaking contracts require explicit service integration evidence.", definition.ID, contractID))
			}
			if entry.Compatibility == contract.Breaking && entry.Kind == contract.Data && (!definition.Evidence.MigrationRequired || !definition.Evidence.RollbackRequired) {
				issues = append(issues, integrationItem("integrate.breaking-data-evidence", "Breaking data contracts require migration and rollback evidence.", definition.ID, contractID))
			}
		}
	}
	return normalizeIntegrationItems(issues)
}
