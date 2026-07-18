package integration_test

import (
	"testing"

	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/integration"
	"fullstack-orchestrator/cli/internal/work"
	"github.com/stretchr/testify/require"
)

func TestCoordinatedContractRequiresProviderBeforeConsumer(t *testing.T) {
	definition := integrationDefinition()
	definition.MergeOrder = []string{"workspace.frontend", "workspace.backend"}
	registry := contract.Registry{SchemaVersion: 1, Contracts: []contract.Entry{{ID: "contract.interface.accounts", Kind: contract.Interface, Status: contract.Approved, Compatibility: contract.Coordinated, Providers: []string{"workspace.backend"}, Consumers: []string{"workspace.frontend"}}}}

	issues := integration.CheckCompatibility([]work.Definition{definition}, registry)
	require.Contains(t, integrationCodes(issues), "integrate.compatibility-order")

	definition.MergeOrder = []string{"workspace.backend", "workspace.frontend"}
	require.Empty(t, integration.CheckCompatibility([]work.Definition{definition}, registry))
}

func TestBreakingDataContractRequiresMigrationAndRollback(t *testing.T) {
	definition := integrationDefinition()
	definition.Evidence.IntegrationRequired = true
	registry := contract.Registry{SchemaVersion: 1, Contracts: []contract.Entry{{ID: "contract.interface.accounts", Kind: contract.Data, Status: contract.Approved, Compatibility: contract.Breaking, Providers: []string{"workspace.backend"}, Consumers: []string{"workspace.frontend"}}}}

	issues := integration.CheckCompatibility([]work.Definition{definition}, registry)
	require.Contains(t, integrationCodes(issues), "integrate.breaking-data-evidence")

	definition.Evidence.MigrationRequired, definition.Evidence.RollbackRequired = true, true
	require.Empty(t, integration.CheckCompatibility([]work.Definition{definition}, registry))
}
