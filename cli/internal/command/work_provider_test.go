package command_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/command"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestProviderReconcileCommitsOnlyStableMappingAndKeepsObservationLocal(t *testing.T) {
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
	observation := filepath.Join(root, ".harness", "local", "providers", "jira", definition.ID+".yaml")
	require.FileExists(t, observation)
	observationData, err := os.ReadFile(observation)
	require.NoError(t, err)
	require.Contains(t, string(observationData), "status: ready")
	require.Contains(t, string(observationData), "source: connector-live")
}

func TestExternalProviderStartUsesFreshAssignmentAndGitSemanticReservation(t *testing.T) {
	root, remote := commandSharedRemote(t)
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "adopt", "--root", root, "--id", "project.external-provider", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\nremote: origin\ncoordination_branch: coordination\n"), 0o600))
	defineCommandWork(t, root, "work.account-recovery", "services/identity/**")
	definition, found, err := loadDefinitionFixture(root, "work.account-recovery")
	require.NoError(t, err)
	require.True(t, found)

	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "github", ItemID: "42", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{
		SchemaVersion: 1, Provider: "github", ItemID: "42", Revision: "etag-42", Status: "in_progress", Owner: "alex", Dependencies: []string{},
		Capabilities: provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "advisory", Revision: true}, DefinitionFingerprint: definition.Fingerprint,
		FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("c", 64),
	}
	reconcileProviderFixture(t, root, mapping, snapshot)
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: configure task tracking")
	commandGit(t, root, "push")

	var output bytes.Buffer
	start := command.New("1.0.0", &output, &bytes.Buffer{})
	start.SetArgs([]string{"work", "start", "--root", root, "--work-id", definition.ID, "--claim-id", "claim.account-recovery", "--owner", "alex", "--branch", "feature/account-recovery", "--apply", "--json"})
	require.NoError(t, start.Execute())
	require.Equal(t, 0, command.ExitCode(start), output.String())
	require.Contains(t, output.String(), "provider.live-revision")
	require.Contains(t, output.String(), "coordination.semantic-reservation")
	require.Contains(t, output.String(), "provider.assignment-advisory")

	observed, err := provider.NewGitLocalStore(root, remote, "coordination").Read(context.Background())
	require.NoError(t, err)
	require.Len(t, observed.Claims, 1)
	require.Equal(t, definition.ID, observed.Claims[0].WorkID)
	require.Equal(t, definition.Fingerprint, observed.Claims[0].DefinitionFingerprint)
	require.Equal(t, "alex", observed.Claims[0].Owner)
	require.FileExists(t, filepath.Join(root, ".harness", "work", "claims", "claim.account-recovery.yaml"))
}

func TestExternalProviderStartRejectsAssignmentForAnotherOwner(t *testing.T) {
	root, _ := commandSharedRemote(t)
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "adopt", "--root", root, "--id", "project.external-owner", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: jira\nlive_status_source: jira\nremote: origin\ncoordination_branch: coordination\n"), 0o600))
	defineCommandWork(t, root, "work.account-recovery", "services/identity/**")
	definition, found, err := loadDefinitionFixture(root, "work.account-recovery")
	require.NoError(t, err)
	require.True(t, found)
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "jira", ItemID: "JIRA-42", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{SchemaVersion: 1, Provider: "jira", ItemID: "JIRA-42", Revision: "42", Status: "in_progress", Owner: "sam", Dependencies: []string{}, Capabilities: provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "verified", Revision: true}, DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("d", 64)}
	reconcileProviderFixture(t, root, mapping, snapshot)
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: configure task tracking")
	commandGit(t, root, "push")

	var output bytes.Buffer
	start := command.New("1.0.0", &output, &bytes.Buffer{})
	start.SetArgs([]string{"work", "start", "--root", root, "--work-id", definition.ID, "--claim-id", "claim.account-recovery", "--owner", "alex", "--branch", "feature/account-recovery", "--apply", "--json"})
	require.NoError(t, start.Execute())
	require.NotEqual(t, 0, command.ExitCode(start), output.String())
	require.Contains(t, output.String(), "provider.owner-mismatch")
}

func TestWorkNextUsesFreshExternalObservationsWithoutTreatingThemAsCanonicalFiles(t *testing.T) {
	root := providerProject(t)
	definition := providerDefinition()
	plan, err := work.PlanDefinition(context.Background(), root, definition)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	definition.Fingerprint = work.Fingerprint(definition)
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "jira", ItemID: "JIRA-123", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{SchemaVersion: 1, Provider: "jira", ItemID: "JIRA-123", Revision: "43", Status: "ready", Dependencies: []string{}, Capabilities: provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "verified", Revision: true}, DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("e", 64)}
	reconcileProviderFixture(t, root, mapping, snapshot)

	var output bytes.Buffer
	next := command.New("1.0.0", &output, &bytes.Buffer{})
	next.SetArgs([]string{"work", "next", "--root", root, "--json"})
	require.NoError(t, next.Execute())
	require.Equal(t, 0, command.ExitCode(next), output.String())
	require.Contains(t, output.String(), "work.recommended")
	require.Contains(t, output.String(), definition.ID)
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

func reconcileProviderFixture(t *testing.T, root string, mapping provider.Mapping, snapshot provider.Snapshot) {
	t.Helper()
	mappingPath := writeProviderFixture(t, "mapping.yaml", mapping)
	snapshotPath := writeProviderFixture(t, "snapshot.yaml", snapshot)
	var output bytes.Buffer
	reconcile := command.New("1.0.0", &output, &bytes.Buffer{})
	reconcile.SetArgs([]string{"work", "provider", "reconcile", "--root", root, "--mapping", mappingPath, "--snapshot", snapshotPath, "--apply", "--json"})
	require.NoError(t, reconcile.Execute())
	require.Equal(t, 0, command.ExitCode(reconcile), output.String())
}

func loadDefinitionFixture(root, id string) (work.Definition, bool, error) {
	definitions, err := work.LoadDefinitions(root)
	if err != nil {
		return work.Definition{}, false, err
	}
	for _, definition := range definitions {
		if definition.ID == id {
			return definition, true, nil
		}
	}
	return work.Definition{}, false, nil
}
