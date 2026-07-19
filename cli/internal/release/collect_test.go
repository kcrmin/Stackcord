package release_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/evidence"
	"github.com/kcrmin/Stackcord/cli/internal/integration"
	"github.com/kcrmin/Stackcord/cli/internal/release"
	"github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type fixtureEvidenceStore struct{ records []evidence.Record }

func (store *fixtureEvidenceStore) Load(string) ([]evidence.Record, error) {
	return append([]evidence.Record(nil), store.records...), nil
}

type fixtureProviderReader struct{ states []integration.ProviderState }

func (reader fixtureProviderReader) Read(context.Context, string, []work.Definition) ([]integration.ProviderState, []domain.Item) {
	return append([]integration.ProviderState(nil), reader.states...), nil
}

func TestCollectedReleaseInputRejectsPointerMismatchAndStaleEvidence(t *testing.T) {
	fixture := newReleaseFixture(t)
	store := &fixtureEvidenceStore{}
	options := release.CollectOptions{Version: "1.0.0", Profile: release.ProfileCore, EvidenceStore: store, ProviderReader: fixtureProviderReader{states: fixture.providerStates}, WorkIDs: []string{"work.release"}}
	initial, _ := release.CollectInput(context.Background(), fixture.root, options)
	store.records = fixture.records(initial.ContractFingerprint)
	store.records = append(store.records, evidence.Record{SchemaVersion: 1, ID: "evidence.999999999999999999999999", Kind: "test", WorkID: "work.previous-release", WorkspaceID: "workspace.root", StartedAt: time.Now().Add(-time.Hour), FinishedAt: time.Now(), ExitCode: 0, Commit: strings.Repeat("9", 40), DefinitionFingerprint: releaseDigest("9"), ContractFingerprint: releaseDigest("8"), OutputDigest: releaseDigest("7")})

	collected, issues := release.CollectInput(context.Background(), fixture.root, options)
	require.Empty(t, issues)
	require.Equal(t, fixture.rootHead, collected.RootCommit)
	require.Equal(t, fixture.childHead, collected.WorkspaceCommits["workspace.backend"])
	require.Equal(t, "https://example.test/backend.git", collected.WorkspaceRemotes["workspace.backend"])

	releaseGit(t, fixture.child, "checkout", "-b", "feature/pointer-drift")
	require.NoError(t, os.WriteFile(filepath.Join(fixture.child, "drift.txt"), []byte("drift\n"), 0o600))
	releaseGit(t, fixture.child, "add", "drift.txt")
	releaseGit(t, fixture.child, "commit", "-m", "feat: change child pointer")
	_, pointerIssues := release.CollectInput(context.Background(), fixture.root, options)
	require.Contains(t, releaseIssueCodes(pointerIssues), "release.pointer-mismatch")

	releaseGit(t, fixture.child, "checkout", fixture.childHead)
	require.NoError(t, os.WriteFile(filepath.Join(fixture.root, "README.md"), []byte("changed\n"), 0o600))
	releaseGit(t, fixture.root, "add", "README.md")
	releaseGit(t, fixture.root, "commit", "-m", "docs: update release notes")
	_, staleIssues := release.CollectInput(context.Background(), fixture.root, options)
	require.Contains(t, releaseIssueCodes(staleIssues), "release.evidence-stale")
}

func TestCollectedReleaseInputIgnoresProcessGitConfigInjection(t *testing.T) {
	fixture := newReleaseFixture(t)
	store := &fixtureEvidenceStore{}
	options := release.CollectOptions{Version: "1.0.0", Profile: release.ProfileCore, EvidenceStore: store, ProviderReader: fixtureProviderReader{states: fixture.providerStates}, WorkIDs: []string{"work.release"}}
	initial, _ := release.CollectInput(context.Background(), fixture.root, options)
	store.records = fixture.records(initial.ContractFingerprint)

	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url.file:///tmp/untrusted-release-rewrite/.insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://example.test/")

	collected, issues := release.CollectInput(context.Background(), fixture.root, options)
	require.Empty(t, issues)
	require.Equal(t, "https://example.test/root.git", collected.WorkspaceRemotes["workspace.root"])
	require.Equal(t, "https://example.test/backend.git", collected.WorkspaceRemotes["workspace.backend"])
}

type releaseFixture struct {
	root, child, rootHead, childHead string
	definition                       work.Definition
	providerStates                   []integration.ProviderState
}

func newReleaseFixture(t *testing.T) releaseFixture {
	t.Helper()
	base := t.TempDir()
	childRemote, rootRemote := filepath.Join(base, "backend.git"), filepath.Join(base, "root.git")
	releaseGit(t, base, "init", "--bare", "--initial-branch=main", childRemote)
	releaseGit(t, base, "init", "--bare", "--initial-branch=main", rootRemote)
	childSource := filepath.Join(base, "backend-source")
	require.NoError(t, os.MkdirAll(childSource, 0o700))
	releaseGit(t, childSource, "init", "-b", "main")
	releaseGitIdentity(t, childSource)
	require.NoError(t, os.WriteFile(filepath.Join(childSource, "backend.txt"), []byte("backend\n"), 0o600))
	releaseGit(t, childSource, "add", ".")
	releaseGit(t, childSource, "commit", "-m", "feat: initialize backend")
	releaseGit(t, childSource, "remote", "add", "origin", childRemote)
	releaseGit(t, childSource, "push", "-u", "origin", "main")

	root := filepath.Join(base, "root")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "definitions"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts", "interfaces"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs"), 0o700))
	releaseGit(t, root, "init", "-b", "main")
	releaseGitIdentity(t, root)
	releaseGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", childRemote, "backend")
	releaseGit(t, filepath.Join(root, "backend"), "remote", "set-url", "origin", "https://example.test/backend.git")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitmodules"), []byte("[submodule \"backend\"]\n\tpath = backend\n\turl = https://example.test/backend.git\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.release-fixture\nlocale: en\n"), 0o600))
	workspaces := "schema_version: 1\nproject_id: project.release-fixture\nroot_remote: https://example.test/root.git\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    remote: https://example.test/root.git\n    responsibilities: [orchestration]\n    dependencies: []\n  - id: workspace.backend\n    kind: submodule\n    path: backend\n    remote: https://example.test/backend.git\n    responsibilities: [backend]\n    dependencies: [workspace.root]\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(workspaces), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "profile.yaml"), []byte("schema_version: 1\ntdd: default\ngit:\n  collaboration: strongly_recommended\n  release: required\ntask_source: git-local\nrelease: core\n"), 0o600))
	contractContent := []byte("---\nschema_version: 1\nid: contract.interface.accounts\nkind: interface\nstatus: approved\nrevision: 1\nrefs: []\n---\n\n# Accounts interface\n\nAccount recovery remains backward compatible.\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "interfaces", "accounts.md"), contractContent, 0o600))
	contractDigest := sha256.Sum256(contractContent)
	registry := "schema_version: 1\ncontracts:\n  - id: contract.interface.accounts\n    kind: interface\n    status: approved\n    revision: 1\n    source: interfaces/accounts.md\n    compatibility: additive\n    providers: [workspace.backend]\n    consumers: []\n    product_ids: []\n    scenario_ids: []\n    data_ids: []\n    ui_ids: []\n    migration_ids: []\n    work_ids: [work.release]\n    test_ids: [test.release]\n    refs: []\n    fingerprint: sha256:" + hex.EncodeToString(contractDigest[:]) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte(registry), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "release.md"), []byte("---\nschema_version: 1\nid: product.release\nkind: product\nstatus: approved\nrevision: 1\nrefs: [contract.interface.accounts]\n---\n\n# Release behavior\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "release.md"), []byte("# Release guide\n"), 0o600))
	definition := work.Definition{SchemaVersion: 1, ID: "work.release", Readiness: work.Ready, Title: "Release", Outcome: "Release service", Acceptance: []work.AcceptanceScenario{{ID: "scenario.release", Given: "ready", When: "released", Then: "works", Failure: "blocks"}}, Refs: []string{"contract.interface.accounts"}, Workspaces: []string{"workspace.backend"}, Scope: work.Scope{Repositories: []string{"repository.backend"}, Paths: []string{"backend/**"}, PolicyIDs: []string{}, ScenarioIDs: []string{"scenario.release"}, ContractIDs: []string{"contract.interface.accounts"}, DBEntities: []string{}, MigrationSlots: []string{}, UIFlows: []string{}, DependencyMajors: []string{}, RootPointers: []string{"workspace.backend"}}, Dependencies: []string{}, MergeOrder: []string{"workspace.backend"}, FirstFailingTest: "test.release", Evidence: work.EvidenceRequirements{Kinds: []string{"test", "child-merge", "root-pointer"}, IntegrationRequired: true}}
	definition.Fingerprint = work.Fingerprint(definition)
	definitionData, err := yaml.Marshal(definition)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "definitions", "work.release.yaml"), definitionData, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("release fixture\n"), 0o600))
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-m", "chore: initialize release fixture")
	releaseGit(t, root, "remote", "add", "origin", rootRemote)
	releaseGit(t, root, "push", "-u", "origin", "main")
	releaseGit(t, root, "remote", "set-url", "origin", "https://example.test/root.git")
	rootHead := releaseGitOutput(t, root, "rev-parse", "HEAD")
	child := filepath.Join(root, "backend")
	childHead := releaseGitOutput(t, child, "rev-parse", "HEAD")
	return releaseFixture{root: root, child: child, rootHead: rootHead, childHead: childHead, definition: definition, providerStates: []integration.ProviderState{{WorkID: definition.ID, Status: "review", Revision: "provider-r1", DefinitionFingerprint: definition.Fingerprint, Confirmed: true}}}
}

func (fixture releaseFixture) records(contractFingerprint string) []evidence.Record {
	now := time.Now().UTC()
	base := evidence.Record{SchemaVersion: 1, WorkID: fixture.definition.ID, DefinitionFingerprint: fixture.definition.Fingerprint, ContractFingerprint: contractFingerprint, StartedAt: now.Add(-time.Second), FinishedAt: now, ExitCode: 0, OutputDigest: releaseDigest("d")}
	test := base
	test.ID, test.Kind, test.WorkspaceID, test.Commit = "evidence.111111111111111111111111", "test", "workspace.backend", fixture.childHead
	test.ArtifactDigests = map[string]string{"service": releaseDigest("a")}
	child := base
	child.ID, child.Kind, child.WorkspaceID, child.Commit = "evidence.222222222222222222222222", "child-merge", "workspace.backend", fixture.childHead
	root := base
	root.ID, root.Kind, root.WorkspaceID, root.Commit = "evidence.333333333333333333333333", "root-pointer", "workspace.root", fixture.rootHead
	return []evidence.Record{test, child, root}
}

func releaseDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }

func releaseIssueCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}

func releaseGitIdentity(t *testing.T, root string) {
	t.Helper()
	releaseGit(t, root, "config", "user.name", "Release Test")
	releaseGit(t, root, "config", "user.email", "release@example.test")
}

func releaseGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_ALLOW_PROTOCOL=file")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func releaseGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}
