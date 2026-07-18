package release

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/provider"
	"fullstack-orchestrator/cli/internal/work"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSelectedProviderReaderBindsExternalRevisionToSemanticReservation(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	root := filepath.Join(base, "root")
	releaseReaderGit(t, base, "init", "--bare", "--initial-branch=main", remote)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "mappings"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "local", "providers", "github"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\nremote: origin\ncoordination_branch: coordination\n"), 0o600))
	definition := work.Definition{SchemaVersion: 1, ID: "work.release", Readiness: work.Ready, Title: "Release", Outcome: "Release safely", Acceptance: []work.AcceptanceScenario{{ID: "scenario.release", Given: "ready", When: "released", Then: "works", Failure: "blocks"}}, Workspaces: []string{"workspace.root"}, Scope: work.Scope{Repositories: []string{"repository.root"}, Paths: []string{"service/**"}}, MergeOrder: []string{"workspace.root"}, FirstFailingTest: "test.release", Evidence: work.EvidenceRequirements{Kinds: []string{"test"}}}
	definition.Fingerprint = work.Fingerprint(definition)
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "github", ItemID: "42", DependencyItems: map[string]string{}}
	observation := provider.Snapshot{SchemaVersion: 1, Provider: "github", ItemID: "42", Revision: "etag-42", Status: "review", Owner: "alex", Dependencies: []string{}, Capabilities: provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "advisory", Revision: true}, DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("a", 64)}
	for path, value := range map[string]any{
		filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml"):             mapping,
		filepath.Join(root, ".harness", "local", "providers", "github", definition.ID+".yaml"): observation,
	} {
		data, err := yaml.Marshal(value)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0o600))
	}
	releaseReaderGit(t, root, "init", "--initial-branch=main")
	releaseReaderGit(t, root, "config", "user.name", "Test User")
	releaseReaderGit(t, root, "config", "user.email", "test@example.com")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".harness/local/\n"), 0o600))
	releaseReaderGit(t, root, "add", ".")
	releaseReaderGit(t, root, "commit", "-m", "chore: initialize release provider fixture")
	releaseReaderGit(t, root, "remote", "add", "origin", remote)
	releaseReaderGit(t, root, "push", "-u", "origin", "main")
	store := provider.NewGitLocalStore(root, "origin", "coordination")
	initial, err := store.Read(context.Background())
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = store.CompareAndSwap(context.Background(), initial.Revision, provider.SnapshotSet{SchemaVersion: 1, Claims: []provider.GitLocalClaim{{ID: "claim.release", WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Status: "review", Owner: "alex", Branch: "feature/release", Repository: "repository.root", Paths: []string{"service/**"}, StartsAt: now, ExpiresAt: now.Add(time.Hour)}}})
	require.NoError(t, err)

	states, issues := (selectedProviderReader{}).Read(context.Background(), root, []work.Definition{definition})

	require.Empty(t, issues)
	require.Len(t, states, 1)
	require.True(t, states[0].Confirmed)
	require.Equal(t, "review", states[0].Status)
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, states[0].Revision)
}

func releaseReaderGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
