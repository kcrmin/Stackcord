package continuity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	workpkg "github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestCollectDistinguishesUnknownAndLocalOnlyEvidence(t *testing.T) {
	root := continuityFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "local-only.txt"), []byte("local\n"), 0o600))

	got := Collect(context.Background(), root, Options{})

	require.Equal(t, "project.example", got.ProjectID)
	require.Equal(t, Unknown, got.Overall)
	require.Contains(t, issueCodes(got.Issues), "provider.live-unknown")
	require.Contains(t, issueCodes(got.Issues), "workspace.local-only")
	require.Contains(t, issueCodes(got.Issues), "workspace.dirty")
	require.Equal(t, "disabled", string(got.Governance.Status))
	require.NotEmpty(t, got.Governance.ProtectedFingerprint)
	require.Len(t, got.NextActions, 1)
}

func TestWorkspaceCollectionBlocksPointerMismatch(t *testing.T) {
	root := t.TempDir()
	manifest := workspace.Manifest{ProjectID: "project.example", Workspaces: []workspace.Entry{
		{ID: "workspace.root", Kind: "root", Path: "."},
		{ID: "workspace.backend", Kind: "submodule", Path: "backend"},
	}}
	rootGit := gitx.State{Root: root, Head: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Submodules: []gitx.Submodule{{
		Path: "backend", Initialized: true, PointerDiff: true,
		ExpectedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Head:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}}

	_, issues := collectWorkspaceStates(context.Background(), root, manifest, rootGit)

	require.Contains(t, issueCodes(issues), "workspace.pointer-mismatch")
}

func TestDetachedSubmoduleCheckoutIsRecoverableFromRootPointer(t *testing.T) {
	entry := workspace.Entry{ID: "workspace.backend", Kind: "submodule", Path: "backend"}
	state := gitx.State{
		Root:     "/project/backend",
		Head:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Detached: true,
	}

	issues := evaluateGitState(entry, state)

	require.NotContains(t, issueCodes(issues), "workspace.local-only")
	require.NotContains(t, issueCodes(issues), "workspace.detached")
}

func TestCollectRecoversGitLocalOwnersAndBranchesAfterClone(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	root := filepath.Join(base, "root")
	clone := filepath.Join(base, "clone")
	runGit(t, base, "init", "--bare", "--initial-branch=main", remote)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "definitions"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: git-local\nlive_status_source: git-local\nremote: origin\ncoordination_branch: coordination\n"), 0o600))
	definition := workpkg.Definition{
		SchemaVersion: 1, ID: "work.account-recovery", Readiness: workpkg.Ready, Title: "Account recovery", Outcome: "Members recover access safely.",
		Acceptance: []workpkg.AcceptanceScenario{{ID: "scenario.account-recovery", Given: "A locked member", When: "Recovery succeeds", Then: "Access is restored", Failure: "Invalid proof is rejected"}},
		Workspaces: []string{"workspace.root"}, Scope: workpkg.Scope{Repositories: []string{"repository.root"}, Paths: []string{"services/recovery/**"}},
		MergeOrder: []string{"workspace.root"}, FirstFailingTest: "test.account-recovery", Evidence: workpkg.EvidenceRequirements{Kinds: []string{"test"}},
	}
	definition.Fingerprint = workpkg.Fingerprint(definition)
	definitionData, err := yaml.Marshal(definition)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "definitions", "work.account-recovery.yaml"), definitionData, 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte("schema_version: 1\ncontracts: []\n"), 0o600))

	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "chore: initialize project")
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "-u", "origin", "main")

	store := provider.NewGitLocalStore(root, "origin", "coordination")
	initial, err := store.Read(context.Background())
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = store.CompareAndSwap(context.Background(), initial.Revision, provider.SnapshotSet{SchemaVersion: 1, Claims: []provider.GitLocalClaim{{
		ID: "claim.account-recovery", WorkID: "work.account-recovery", DefinitionFingerprint: definition.Fingerprint, Status: "in_progress",
		Owner: "alex", Branch: "feature/account-recovery", Repository: "repository.root", Paths: []string{"services/recovery/**"},
		StartsAt: now, ExpiresAt: now.Add(time.Hour),
	}}})
	require.NoError(t, err)
	runGit(t, base, "clone", remote, clone)

	got := Collect(context.Background(), clone, Options{})

	require.Equal(t, Confirmed, got.Provider.Confidence)
	require.Len(t, got.ActiveWork, 1)
	require.Equal(t, "alex", got.ActiveWork[0].Owner)
	require.Equal(t, "feature/account-recovery", got.ActiveWork[0].Branch)
	require.Equal(t, "in_progress", got.ActiveWork[0].State)
}

func TestCollectUsesFreshExternalObservationAndRecoversReservationAfterClone(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	root := filepath.Join(base, "root")
	clone := filepath.Join(base, "clone")
	runGit(t, base, "init", "--bare", "--initial-branch=main", remote)
	for _, directory := range []string{
		filepath.Join(root, ".harness", "work", "definitions"),
		filepath.Join(root, ".harness", "work", "mappings"),
		filepath.Join(root, ".harness", "local", "providers", "github"),
		filepath.Join(root, "specs"), filepath.Join(root, "contracts"), filepath.Join(root, "docs"),
	} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".harness/local/\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.external\nlocale: en\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.external\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\nremote: origin\ncoordination_branch: coordination\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte("schema_version: 1\ncontracts: []\n"), 0o600))
	definition := workpkg.Definition{
		SchemaVersion: 1, ID: "work.account-recovery", Readiness: workpkg.Ready, Title: "Account recovery", Outcome: "Members recover access safely.",
		Acceptance: []workpkg.AcceptanceScenario{{ID: "scenario.account-recovery", Given: "A locked member", When: "Recovery succeeds", Then: "Access is restored", Failure: "Invalid proof is rejected"}},
		Workspaces: []string{"workspace.root"}, Scope: workpkg.Scope{Repositories: []string{"repository.root"}, Paths: []string{"services/recovery/**"}},
		MergeOrder: []string{"workspace.root"}, FirstFailingTest: "test.account-recovery", Evidence: workpkg.EvidenceRequirements{Kinds: []string{"test"}},
	}
	definition.Fingerprint = workpkg.Fingerprint(definition)
	definitionData, err := yaml.Marshal(definition)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "definitions", definition.ID+".yaml"), definitionData, 0o600))
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "github", ItemID: "42", DependencyItems: map[string]string{}}
	mappingData, err := yaml.Marshal(mapping)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml"), mappingData, 0o600))
	observation := provider.Snapshot{SchemaVersion: 1, Provider: "github", ItemID: "42", Revision: "etag-42", Status: "in_progress", Owner: "alex", Dependencies: []string{}, Capabilities: provider.Capabilities{Hierarchy: true, Dependencies: true, Claim: "advisory", Revision: true}, DefinitionFingerprint: definition.Fingerprint, FetchedAt: time.Now().UTC(), Source: "connector-live", RawHash: "sha256:" + strings.Repeat("a", 64)}
	observationData, err := yaml.Marshal(observation)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "local", "providers", "github", definition.ID+".yaml"), observationData, 0o600))

	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "chore: initialize external provider project")
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "-u", "origin", "main")
	store := provider.NewGitLocalStore(root, "origin", "coordination")
	initial, err := store.Read(context.Background())
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = store.CompareAndSwap(context.Background(), initial.Revision, provider.SnapshotSet{SchemaVersion: 1, Claims: []provider.GitLocalClaim{{ID: "claim.account-recovery", WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Status: "in_progress", Owner: "alex", Branch: "feature/account-recovery", Repository: "repository.root", Paths: []string{"services/recovery/**"}, StartsAt: now, ExpiresAt: now.Add(time.Hour)}}})
	require.NoError(t, err)

	local := Collect(context.Background(), root, Options{})
	require.Equal(t, Confirmed, local.Provider.Confidence)
	require.NotContains(t, issueCodes(local.Issues), "provider.live-unknown")
	require.Len(t, local.ActiveWork, 1)
	require.Equal(t, "alex", local.ActiveWork[0].Owner)
	require.Equal(t, "feature/account-recovery", local.ActiveWork[0].Branch)

	runGit(t, base, "clone", remote, clone)
	recovered := Collect(context.Background(), clone, Options{})
	require.Equal(t, Unknown, recovered.Provider.Confidence, "a clone must re-read the external provider")
	require.Contains(t, issueCodes(recovered.Issues), "provider.live-unknown")
	require.Len(t, recovered.ActiveWork, 1)
	require.Equal(t, "alex", recovered.ActiveWork[0].Owner)
	require.Equal(t, "feature/account-recovery", recovered.ActiveWork[0].Branch)
	require.Equal(t, "in_progress", recovered.ActiveWork[0].State)
}

func continuityFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{
		filepath.Join(root, ".harness", "work"),
		filepath.Join(root, "specs", "policies"),
		filepath.Join(root, "contracts"),
	} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\n"), 0o600))
	policy := "---\nschema_version: 1\nid: policy.example\nkind: policy\nstatus: approved\nrevision: 1\nrefs: []\n---\nExample.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "example.md"), []byte(policy), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte("schema_version: 1\ncontracts: []\n"), 0o600))
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "chore: initialize project")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func issueCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
