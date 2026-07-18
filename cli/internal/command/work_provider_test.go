package command_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/command"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/provider"
	"fullstack-orchestrator/cli/internal/work"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestProviderReconcileCommitsOnlyStableMapping(t *testing.T) {
	root := providerProject(t)
	definition := providerDefinition()
	plan, err := work.PlanDefinition(context.Background(), root, definition)
	require.NoError(t, err)
	require.Empty(t, plan.Blockers)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	definition.Fingerprint = work.Fingerprint(definition)

	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "jira", ItemID: "JIRA-123", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{
		SchemaVersion: 1, Provider: "jira", ItemID: "JIRA-123", Revision: "42", Status: "ready", Dependencies: []string{},
		Capabilities:          provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "verified", Revision: true},
		DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("b", 64),
	}
	mappingPath := writeProviderFixture(t, "mapping.yaml", mapping)
	snapshotPath := writeProviderFixture(t, "snapshot.yaml", snapshot)

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "provider", "reconcile", "--root", root, "--mapping", mappingPath, "--snapshot", snapshotPath, "--apply", "--json"})
	require.NoError(t, cmd.Execute())

	mapped := filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml")
	require.FileExists(t, mapped)
	data, err := os.ReadFile(mapped)
	require.NoError(t, err)
	require.NotContains(t, string(data), "owner:")
	require.NotContains(t, string(data), "status:")
	require.NoFileExists(t, filepath.Join(root, ".harness", "local", "providers", "jira", definition.ID+".json"))
}

func TestProviderReconcileRejectsProviderOtherThanSelectedLiveSource(t *testing.T) {
	root := providerProject(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: git-local\nlive_status_source: git-local\n"), 0o600))
	definition := providerDefinition()
	plan, err := work.PlanDefinition(context.Background(), root, definition)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	definition.Fingerprint = work.Fingerprint(definition)

	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "jira", ItemID: "JIRA-123", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{
		SchemaVersion: 1, Provider: "jira", ItemID: "JIRA-123", Revision: "42", Status: "ready", Dependencies: []string{},
		Capabilities:          provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "verified", Revision: true},
		DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("b", 64),
	}
	mappingPath := writeProviderFixture(t, "mapping.yaml", mapping)
	snapshotPath := writeProviderFixture(t, "snapshot.yaml", snapshot)

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "provider", "reconcile", "--root", root, "--mapping", mappingPath, "--snapshot", snapshotPath, "--apply", "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "provider.selected-mismatch")
	require.NoFileExists(t, filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml"))
}

func providerProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: jira\nlive_status_source: jira\n"), 0o600))
	return root
}

func providerDefinition() work.Definition {
	return work.Definition{
		SchemaVersion: 1, ID: "work.provider-check", Readiness: work.Ready, Title: "Check provider", Outcome: "Provider state is observable.",
		Acceptance: []work.AcceptanceScenario{{ID: "scenario.provider-check", Given: "selected provider work", When: "the connector reads it", Then: "revision and status are normalized", Failure: "stale data is rejected"}},
		Refs:       []string{}, Workspaces: []string{"workspace.root"}, Scope: work.Scope{Repositories: []string{"repository.root"}, Paths: []string{".harness/work/mappings"}},
		Dependencies: []string{}, MergeOrder: []string{"workspace.root"}, FirstFailingTest: "test.provider-reconcile", Evidence: work.EvidenceRequirements{Kinds: []string{"integration"}},
	}
}

func writeProviderFixture(t *testing.T, name string, value any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	data, err := yaml.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}
