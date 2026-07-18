# Service Continuity Production Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the existing deterministic Git and semantic-safety core into a production-quality Codex Plugin and cross-platform CLI that recovers multi-repository service context, coordinates real work ownership, prevents semantic collisions, and verifies exact release candidates.

**Architecture:** Keep `gitx`, `context`, `policy`, `contract`, `database`, `ui`, `operation`, and `release` as focused deterministic cores. Add bounded `continuity`, `workspace`, `work`, `provider`, and `evidence` packages; Skills own natural-language judgment and connector calls while the CLI validates normalized snapshots and actual repository state. Replace current plan-only and string-only paths through tests before deleting them.

**Tech Stack:** Go 1.26.x, Cobra 1.10.2, JSON Schema Draft 2020-12, strict YAML/JSON, Git 2.25+, Codex Agent Skills and Plugin hooks, Python 3 standard-library repository validators, GitHub Actions, GoReleaser, optional external GitHub/Jira/Beads connectors.

## Global Constraints

- Work in `/Users/ryanmin/Documents/dev/Project/fullstack-orchestrator`; do not mix product files or history with Soomgil.
- Preserve branch `chore/pre-refocus-workspace` and commit `33c5d44` as recovery points.
- The AI owns flexible conversation and judgment; the CLI owns deterministic actual-state and identity verification.
- Generated projects remain framework, language, database, cloud, AI-client, and task-provider neutral.
- The default product exposes five Skills and one compact repo-local entry Skill.
- Raw conversation, tone, prompts, credentials, and secrets are never persisted.
- Generated context and provider snapshots are local cache, never canonical shared state.
- Exactly one provider owns live work status; repository work definitions own product meaning.
- Git is optional for early solo discovery, strongly recommended for collaboration, and required for remote recovery and verifiable release.
- Behavior, bug, contract, migration, and UI-interaction changes use red/green TDD.
- Branch, commit, PR, and tag names contain no AI, agent, Codex, GPT, model, or generated-by marker.
- Do not silently pull, rebase, stash, reset, clean, force-push feature history, install external software, write to a provider, or publish.
- Unknown external state remains `unknown`; no cached provider snapshot can prove a live claim.
- Default mode must not require strict-release supply-chain fields.
- macOS arm64/amd64 and Windows arm64/amd64 are release targets; Linux remains supported for development and CI.
- Public name, publisher account, repository URL, signing identity, and irreversible publication remain external decisions.

## File and package map

New bounded units:

- `cli/internal/workspace/`: project-root discovery, workspace manifests, child bridge, and actual root/child snapshots.
- `cli/internal/provider/`: provider capabilities, normalized snapshots, mapping drift, and Git-local compare-and-swap state.
- `cli/internal/work/`: rich definitions, lifecycle, claims, scope changes, and merge ordering.
- `cli/internal/evidence/`: command-bound test/review/integration/user evidence and current-commit verification.
- `cli/internal/continuity/`: combines context, workspace, provider, work, and release state into one resume snapshot and next action.
- `cli/internal/hook/`: converts continuity output into small SessionStart/PostCompact hook payloads without tracked writes.
- `cli/internal/tooling/`: detected-tool facts and dated selection records; it does not search the web itself.

Existing units retained:

- `cli/internal/context/`: canonical stable IDs, fingerprints, stale/unknown, and impact graph.
- `cli/internal/gitx/`: actual Git/submodule/worktree inspection and narrowly scoped safe mutations.
- `cli/internal/policy/`: normalized path and semantic conflict engine.
- `cli/internal/contract/`, `database/`, `ui/`, `release/`, `operation/`: domain validation and atomic local writes.

Command files are split by responsibility instead of extending the current large `cli/internal/command/work.go`:

- `status.go`, `hook.go`, `workspace.go`;
- `work_define.go`, `work_provider.go`, `work_lifecycle.go`;
- existing boundary, Git, integration, and release commands remain small adapters.

---

## Milestone 1 — Trustworthy installation and context foundation

### Task 1: Repair authoritative entrypoints, Plugin hooks, and validation

**Files:**
- Modify: `AGENTS.md`
- Modify: `docs/design/index.md`
- Modify: `.codex-plugin/plugin.json`
- Replace: `hooks/hooks.json`
- Modify: `scripts/validate_plugin.py`
- Modify: `scripts/validate_plugin_test.py`
- Create: `testdata/plugin/hooks-valid.json`
- Create: `testdata/plugin/hooks-invalid.json`

**Interfaces:**
- Produces current `SessionStart` and `PostCompact` command-hook configuration.
- Produces `validate_hook_document(value: object) -> list[str]` in the repository validator.
- Points every product agent to the final design and this plan.

- [x] **Step 1: Add failing tests for current hook structure and live documentation links.**

```python
def test_hooks_use_current_command_schema(self):
    hooks = json.loads((ROOT / "hooks" / "hooks.json").read_text())
    self.assertEqual({"SessionStart", "PostCompact"}, set(hooks["hooks"]))
    for event in hooks["hooks"].values():
        command = event[0]["hooks"][0]
        self.assertEqual("command", command["type"])
        self.assertIn("orchestrator hook", command["command"])

def test_agent_entry_links_exist(self):
    text = (ROOT / "AGENTS.md").read_text()
    for path in re.findall(r"`([^`]+\.md)`", text):
        self.assertTrue((ROOT / path).is_file(), path)
```

- [x] **Step 2: Run the tests and verify failure.**

Run: `python3 scripts/validate_plugin_test.py -v`

Expected: FAIL because hooks are an event list with message-only entries and `AGENTS.md` points to deleted files.

- [x] **Step 3: Replace the hook document with command hooks.**

```json
{
  "hooks": {
    "SessionStart": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "orchestrator hook session-start", "timeout": 10}]
    }],
    "PostCompact": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "orchestrator hook post-compact", "timeout": 10}]
    }]
  }
}
```

Set `.codex-plugin/plugin.json` `hooks` to `./hooks/hooks.json`, as required by the current Codex manual for an explicit bundled lifecycle resource. Codex can also discover the default path, but the manifest keeps published package intent unambiguous. Verify the repository validator and a real Codex plugin ingestion path; do not rely on the older scaffold validator that still rejects this now-supported field. Update `AGENTS.md` to reference:

```text
docs/superpowers/specs/2026-07-18-service-continuity-harness-design.md
docs/superpowers/plans/2026-07-18-service-continuity-production.md
```

- [x] **Step 4: Make the local validator reject the old shape and malformed command hooks.**

```python
def validate_hook_document(value):
    errors = []
    if not isinstance(value, dict) or not isinstance(value.get("hooks"), dict):
        return ["hooks must be an event-keyed object"]
    for name in ("SessionStart", "PostCompact"):
        groups = value["hooks"].get(name)
        if not isinstance(groups, list) or not groups:
            errors.append(f"missing hook event {name}")
            continue
        for group in groups:
            commands = group.get("hooks", [])
            if not commands or any(item.get("type") != "command" or not item.get("command") for item in commands):
                errors.append(f"{name} must contain command hooks")
    return errors
```

- [x] **Step 5: Run repository and official validators.**

Run:

```sh
python3 scripts/validate_plugin_test.py -v
python3 scripts/validate_plugin.py .
```

Expected: PASS locally; the official Plugin validator must also accept the manifest and hook resource before this task is complete.

- [x] **Step 6: Commit.**

```sh
git add AGENTS.md docs/design/index.md .codex-plugin/plugin.json hooks scripts testdata/plugin
git commit -m "fix(plugin): validate lifecycle hooks"
```

### Task 2: Separate canonical project state from local generated state and add root-child bridges

**Files:**
- Create: `cli/internal/workspace/model.go`
- Create: `cli/internal/workspace/load.go`
- Create: `cli/internal/workspace/bridge.go`
- Create: `cli/internal/workspace/workspace_test.go`
- Modify: `cli/internal/project/generate.go`
- Modify: `cli/internal/project/adopt.go`
- Modify: `cli/internal/project/project_e2e_test.go`
- Modify: `cli/internal/context/refresh.go`
- Modify: `cli/internal/context/refresh_test.go`
- Modify: `cli/internal/command/root.go`
- Modify: `cli/internal/command/root_test.go`
- Modify: `templates/project/AGENTS.md.tmpl`
- Modify: `templates/project/.agents/skills/use-project-harness/SKILL.md`
- Modify: `templates/project/.agents/skills/use-project-harness/references/fallback.md`
- Replace: `schemas/workspaces.schema.json`
- Replace: `cli/internal/schema/definitions/workspaces.schema.json`

**Interfaces:**
- Produces `workspace.Manifest`, `workspace.Entry`, and `workspace.Bridge`.
- Produces `workspace.FindRoot(ctx, start) (Root, error)` and `workspace.Load(root) (Manifest, error)`.
- Canonical manifests are committed; `.harness/local/**` is ignored and never indexed.

- [x] **Step 1: Write failing unit and E2E tests.**

```go
func TestFindRootFromSubmoduleUsesActualSuperproject(t *testing.T) {
    fixture := newRootWithChildSubmodule(t)
    root, err := workspace.FindRoot(context.Background(), fixture.Child)
    require.NoError(t, err)
    assert.Equal(t, fixture.Root, root.Path)
    assert.Equal(t, "workspace.backend", root.CurrentWorkspaceID)
}

func TestGeneratedLocalStateIsIgnoredAndAbsent(t *testing.T) {
    root := generateProject(t)
    assert.Contains(t, read(t, root, ".gitignore"), ".harness/local/")
    assert.NoFileExists(t, filepath.Join(root, ".harness", "local", "context", "context-index.json"))
    assert.NoFileExists(t, filepath.Join(root, ".harness", "state", "context-index.json"))
}
```

- [x] **Step 2: Run focused tests and verify failure.**

Run: `cd cli && go test ./internal/workspace ./internal/project -run 'Root|LocalState|Submodule' -v`

Expected: FAIL because `workspace` does not exist and generated context state is currently tracked.

- [x] **Step 3: Define focused models.**

```go
type Entry struct {
    ID             string   `json:"id" yaml:"id"`
    Kind           string   `json:"kind" yaml:"kind"`
    Path           string   `json:"path" yaml:"path"`
    Repository     string   `json:"repository,omitempty" yaml:"repository,omitempty"`
    Remote         string   `json:"remote,omitempty" yaml:"remote,omitempty"`
    Responsibilities []string `json:"responsibilities" yaml:"responsibilities"`
    Dependencies   []string `json:"dependencies" yaml:"dependencies"`
    ContractFingerprint string `json:"contract_fingerprint,omitempty" yaml:"contract_fingerprint,omitempty"`
    CommandsPath   string   `json:"commands_path,omitempty" yaml:"commands_path,omitempty"`
}

type Manifest struct {
    SchemaVersion int     `json:"schema_version" yaml:"schema_version"`
    ProjectID     string  `json:"project_id" yaml:"project_id"`
    RootRemote    string  `json:"root_remote,omitempty" yaml:"root_remote,omitempty"`
    Workspaces    []Entry `json:"workspaces" yaml:"workspaces"`
}

type Bridge struct {
    SchemaVersion int    `json:"schema_version" yaml:"schema_version"`
    ProjectID     string `json:"project_id" yaml:"project_id"`
    RootRemote    string `json:"root_remote" yaml:"root_remote"`
    WorkspaceID   string `json:"workspace_id" yaml:"workspace_id"`
    Discovery     string `json:"discovery" yaml:"discovery"`
    ContractFingerprint string `json:"contract_fingerprint" yaml:"contract_fingerprint"`
    CommandsPath  string `json:"commands_path" yaml:"commands_path"`
}

type State struct {
    Entry       Entry      `json:"entry"`
    Git         gitx.State `json:"git"`
    ExpectedSHA string     `json:"expected_sha,omitempty"`
    Confidence  string     `json:"confidence"`
    Issues      []domain.Item `json:"issues"`
}
```

- [x] **Step 4: Implement root discovery and bridge validation.**

Use `git rev-parse --show-superproject-working-tree` first, then walk ancestors for `.harness/project.yaml`. Reject project-ID, root-remote, workspace-ID, and contract-fingerprint disagreement. A child-only clone returns a typed incomplete-context issue rather than success.

- [x] **Step 5: Replace generated state with canonical manifests and local ignore rules.**

Generated files retain the compact `AGENTS.md`, repo-local Skill, project/profile/source/workspace manifests, specs index, contract registry, and docs index. Remove tracked generated context index and impact graph from new projects; regenerate them under `.harness/local/context/`.

- [x] **Step 6: Verify non-destructive adoption.**

Run: `cd cli && go test -race ./internal/workspace ./internal/project -v`

Expected: PASS, including preservation of existing README, AGENTS sections, and authored specs/contracts.

- [x] **Step 7: Commit.**

```sh
git add cli/internal/workspace cli/internal/project templates/project schemas cli/internal/schema/definitions
git commit -m "feat(project): connect root and child workspaces"
```

### Task 3: Build one combined continuity status and hook packet

**Files:**
- Create: `cli/internal/continuity/model.go`
- Create: `cli/internal/continuity/collect.go`
- Create: `cli/internal/continuity/next.go`
- Create: `cli/internal/continuity/continuity_test.go`
- Create: `cli/internal/hook/render.go`
- Create: `cli/internal/hook/render_test.go`
- Create: `cli/internal/command/status.go`
- Create: `cli/internal/command/hook.go`
- Modify: `cli/internal/command/root.go`
- Modify: `cli/internal/command/root_test.go`
- Modify: `cli/internal/command/project.go`
- Modify: `cli/internal/command/git.go`

**Interfaces:**
- Produces `continuity.Collect(ctx, root, Options) Snapshot`.
- Produces `hook.Render(event string, snapshot continuity.Snapshot) ([]byte, error)`.
- Adds `orchestrator status --root --json` and `orchestrator hook session-start|post-compact`.

- [x] **Step 1: Write failing combined-state tests.**

```go
func TestCollectDistinguishesConfirmedStaleUnknownAndLocalOnly(t *testing.T) {
    fixture := newContinuityFixture(t)
    fixture.Backend.SetPointerMismatch()
    fixture.Provider.WriteCachedOnlySnapshot()
    fixture.Frontend.CreateUnpushedCommit()

    got := continuity.Collect(context.Background(), fixture.Root, continuity.Options{})

    assert.Equal(t, continuity.Blocked, got.Overall)
    assert.Contains(t, codes(got.Issues), "workspace.pointer-mismatch")
    assert.Contains(t, codes(got.Issues), "provider.live-unknown")
    assert.Contains(t, codes(got.Issues), "workspace.local-only")
    assert.Len(t, got.NextActions, 1)
}
```

- [x] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/continuity ./internal/hook ./internal/command -run 'Continuity|Status|Hook' -v`

Expected: FAIL because packages and commands do not exist.

- [x] **Step 3: Define the snapshot.**

```go
type Confidence string
const (
    Confirmed Confidence = "confirmed"
    Warning   Confidence = "warning"
    Stale     Confidence = "stale"
    Unknown   Confidence = "unknown"
    LocalOnly Confidence = "local-only"
    Blocked   Confidence = "blocked"
)

type Snapshot struct {
    ProjectID string              `json:"project_id"`
    Overall   Confidence          `json:"overall"`
    Context   contextpkg.Snapshot `json:"context"`
    Workspaces []workspace.State  `json:"workspaces"`
    Provider  ProviderView        `json:"provider"`
    ActiveWork []WorkView         `json:"active_work"`
    Release   ReleaseView         `json:"release"`
    Issues    []domain.Item       `json:"issues"`
    NextActions []domain.Item     `json:"next_actions"`
}

type ProviderView struct {
    Name, ItemID, Revision, Owner, Status, Confidence string
}

type WorkView struct {
    ID, Title, State, DefinitionFingerprint string
}

type ReleaseView struct {
    CandidateDigest, Confidence string
}
```

- [x] **Step 4: Implement collection in evidence order.**

Collect actual Git/workspaces first, approved canonical context second, currently observable provider/work files third, release state next, and generated cache last. The initial provider view reports external state as unknown until Task 5 supplies a freshly reconciled snapshot. Task 5 and Task 8 populate the stable `ProviderView` and `WorkView` without changing the status JSON contract. Do not let missing optional cache lower confirmed source evidence. Sort all outputs for stable JSON.

- [x] **Step 5: Render compact hooks without writes.**

The `SessionStart` JSON uses `hookSpecificOutput.additionalContext` and includes project ID, canonical fingerprint, active work IDs, related source paths, blocker codes/refs, and one next action. `PostCompact` uses only the supported `systemMessage`; the following `SessionStart` with source `compact` injects the packet. Output excludes full documents, issue free text, raw Git status, and credentials. If the CLI is unavailable the hook command may fail; Skills must still run the repo-local preflight.

- [x] **Step 6: Upgrade doctor to report actionable tool readiness.**

Add Git version, root discovery, CLI path/version, selected provider type, connector availability facts supplied by local detection, dbdiagram presence, and reduced-verification warnings. Detection remains read-only and does not install.

- [x] **Step 7: Run tests.**

Run: `cd cli && go test -race ./internal/continuity ./internal/hook ./internal/command ./internal/workspace ./internal/gitx ./internal/context -v`

Expected: PASS with no tracked `.harness` mutation from status or hooks.

- [x] **Step 8: Commit.**

```sh
git add cli/internal/continuity cli/internal/hook cli/internal/command
git commit -m "feat(context): combine project continuity status"
```

---

## Milestone 2 — Real work lifecycle and coordination

### Task 4: Replace shallow work items with executable work definitions

**Files:**
- Create: `cli/internal/work/model.go`
- Create: `cli/internal/work/definition.go`
- Create: `cli/internal/work/definition_test.go`
- Create: `cli/internal/command/work_define.go`
- Modify: `cli/internal/domain/work.go`
- Replace: `schemas/work-item.schema.json`
- Replace: `cli/internal/schema/definitions/work-item.schema.json`
- Modify: `cli/internal/project/generate.go`
- Modify: `cli/internal/command/root.go`
- Split: `cli/internal/command/work.go`

**Interfaces:**
- Produces `work.Definition`, `work.Scope`, `work.EvidenceRequirements`.
- Produces `work.PlanDefinition(root string, definition Definition) (operation.Plan, error)`.
- Adds `work define --input [--apply]` while preserving read compatibility long enough to migrate current fixtures.

- [x] **Step 1: Write failing definition tests.**

```go
func TestReadyDefinitionRequiresAcceptanceScopeOrderAndFirstTest(t *testing.T) {
    definition := validDefinition()
    definition.Acceptance = nil
    definition.FirstFailingTest = ""
    issues := work.ValidateDefinition(definition)
    assert.ElementsMatch(t, []string{"work.acceptance-required", "work.first-test-required"}, codes(issues))
}

func TestDefinitionFingerprintChangesWhenSemanticScopeExpands(t *testing.T) {
    before := validDefinition()
    after := before
    after.Scope.DBEntities = append(after.Scope.DBEntities, "account_recovery")
    assert.NotEqual(t, work.Fingerprint(before), work.Fingerprint(after))
}
```

- [x] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/work ./internal/command -run 'Definition|WorkDefine' -v`

Expected: FAIL because the rich definition does not exist.

- [x] **Step 3: Define the model.**

```go
type Definition struct {
    SchemaVersion int                  `json:"schema_version" yaml:"schema_version"`
    ID            string               `json:"id" yaml:"id"`
    ParentID      string               `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
    Title         string               `json:"title" yaml:"title"`
    Outcome       string               `json:"outcome" yaml:"outcome"`
    Acceptance    []AcceptanceScenario `json:"acceptance" yaml:"acceptance"`
    Refs          []string             `json:"refs" yaml:"refs"`
    Workspaces    []string             `json:"workspaces" yaml:"workspaces"`
    Scope         Scope                `json:"scope" yaml:"scope"`
    Dependencies  []string             `json:"dependencies" yaml:"dependencies"`
    MergeOrder    []string             `json:"merge_order" yaml:"merge_order"`
    FirstFailingTest string             `json:"first_failing_test" yaml:"first_failing_test"`
    Evidence      EvidenceRequirements `json:"evidence" yaml:"evidence"`
    Fingerprint   string               `json:"fingerprint" yaml:"fingerprint"`
}
```

`Scope` includes repository/workspace, paths, policy IDs, scenario IDs, contract IDs, DB entities, migration slots, UI flows, dependency majors, and root-pointer ownership.

- [x] **Step 4: Implement strict validation and automatic fingerprinting.**

Reject duplicate IDs, missing referenced canonical entries, cyclic dependencies, merge-order items not present in the work graph, ready definitions without actionable acceptance, and user-supplied fingerprints that differ from normalized content.

- [x] **Step 5: Add plan/apply command and migrate fixtures.**

The command reads strict YAML/JSON and writes `.harness/work/definitions/<id>.yaml` atomically. Live status and owner are absent from the file.

- [x] **Step 6: Run tests and commit.**

```sh
cd cli && go test -race ./internal/work ./internal/command ./internal/project ./internal/schema -v
cd ..
git add cli schemas templates examples testdata
git commit -m "feat(work): define executable project work"
```

### Task 5: Add provider mappings, normalized snapshots, and drift reconciliation

**Files:**
- Create: `cli/internal/provider/model.go`
- Create: `cli/internal/provider/load.go`
- Create: `cli/internal/provider/reconcile.go`
- Create: `cli/internal/provider/provider_test.go`
- Create: `cli/internal/command/work_provider.go`
- Create: `schemas/provider-mapping.schema.json`
- Create: `schemas/provider-snapshot.schema.json`
- Copy: `cli/internal/schema/definitions/provider-mapping.schema.json`
- Copy: `cli/internal/schema/definitions/provider-snapshot.schema.json`
- Modify: `cli/internal/schema/loader.go`
- Modify: `cli/internal/continuity/collect.go`

**Interfaces:**
- Produces `provider.Mapping`, `provider.Expectation`, `provider.Snapshot`, `provider.State`, and `provider.Capabilities` without importing the `work` package.
- Produces `provider.Reconcile(expectation Expectation, mapping Mapping, snapshot Snapshot, now time.Time) State`.
- Adds `work provider reconcile --mapping --snapshot` and stable result codes.

- [ ] **Step 1: Write failing freshness, drift, and capability tests.**

```go
func TestReconcileRejectsCachedOrDriftedProviderState(t *testing.T) {
    definition := validDefinition()
    mapping := validMapping(definition)
    snapshot := validSnapshot(mapping)
    snapshot.FetchedAt = time.Now().Add(-25 * time.Hour)
    snapshot.DefinitionFingerprint = "sha256:" + strings.Repeat("0", 64)

    expectation := provider.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}
    state := provider.Reconcile(expectation, mapping, snapshot, time.Now())

    assert.Equal(t, provider.Unknown, state.Confidence)
    assert.Contains(t, codes(state.Issues), "provider.snapshot-stale")
    assert.Contains(t, codes(state.Issues), "provider.definition-drift")
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/provider ./internal/continuity ./internal/command -run Provider -v`

Expected: FAIL because provider contracts do not exist.

- [ ] **Step 3: Define strict provider data.**

```go
type Capabilities struct {
    Hierarchy bool   `json:"hierarchy" yaml:"hierarchy"`
    Dependencies bool `json:"dependencies" yaml:"dependencies"`
    Claim string      `json:"claim" yaml:"claim"` // atomic, verified, advisory, none
    Revision bool     `json:"revision" yaml:"revision"`
}

type Expectation struct {
    WorkID               string
    DefinitionFingerprint string
    Dependencies         []string
}

type Snapshot struct {
    SchemaVersion int          `json:"schema_version" yaml:"schema_version"`
    Provider      string       `json:"provider" yaml:"provider"`
    ItemID        string       `json:"item_id" yaml:"item_id"`
    Revision      string       `json:"revision" yaml:"revision"`
    Status        string       `json:"status" yaml:"status"`
    Owner         string       `json:"owner,omitempty" yaml:"owner,omitempty"`
    Dependencies  []string     `json:"dependencies" yaml:"dependencies"`
    Capabilities  Capabilities `json:"capabilities" yaml:"capabilities"`
    DefinitionFingerprint string `json:"definition_fingerprint" yaml:"definition_fingerprint"`
    FetchedAt     time.Time    `json:"fetched_at" yaml:"fetched_at"`
    Source        string       `json:"source" yaml:"source"`
    RawHash       string       `json:"raw_hash" yaml:"raw_hash"`
}
```

- [ ] **Step 4: Implement reconciliation.**

Validate selected-provider identity, item ID, revision availability, current definition fingerprint, known normalized status, owner semantics, dependency mapping, freshness policy, source provenance, and raw hash. A local file alone never upgrades an external snapshot to live.

- [ ] **Step 5: Add local-only snapshot placement.**

Provider connector output is accepted only from `.harness/local/providers/` or an explicit temporary input path. The CLI writes only the stable mapping to `.harness/work/mappings/`.

- [ ] **Step 6: Run tests and commit.**

```sh
cd cli && go test -race ./internal/provider ./internal/continuity ./internal/command ./internal/schema -v
cd ..
git add cli/internal/provider cli/internal/continuity cli/internal/command cli/internal/schema schemas
git commit -m "feat(work): reconcile live task providers"
```

### Task 6: Implement race-detecting Git-local coordination

**Files:**
- Create: `cli/internal/provider/gitlocal.go`
- Create: `cli/internal/provider/gitlocal_test.go`
- Modify: `cli/internal/gitx/runner.go`
- Delete after replacement: `cli/internal/gitx/remote_files.go`
- Modify: `cli/internal/command/work_provider.go`
- Create: `cli/internal/command/gitlocal_e2e_test.go`

**Interfaces:**
- Produces `provider.GitLocalStore` with `Read(ctx) (SnapshotSet, error)` and `CompareAndSwap(ctx, expected string, next SnapshotSet) (string, error)`.
- Uses remote branch `coordination` by default and an isolated temporary index/worktree; it never checks out the coordination branch in the user's worktree.

- [ ] **Step 1: Write an E2E race test with a bare remote and two clones.**

```go
func TestGitLocalCompareAndSwapAllowsOneConcurrentClaim(t *testing.T) {
    remote, left, right := newSharedRemote(t)
    leftStore := provider.NewGitLocalStore(left, remote, "coordination")
    rightStore := provider.NewGitLocalStore(right, remote, "coordination")
    base := seedCoordination(t, leftStore)

    leftResult := make(chan error, 1)
    rightResult := make(chan error, 1)
    go func() { _, err := leftStore.CompareAndSwap(context.Background(), base, claimedBy("left")); leftResult <- err }()
    go func() { _, err := rightStore.CompareAndSwap(context.Background(), base, claimedBy("right")); rightResult <- err }()

    errors := []error{<-leftResult, <-rightResult}
    assert.Equal(t, 1, countNil(errors))
    assert.Equal(t, 1, countCASConflict(errors))
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/provider ./internal/command -run GitLocal -v`

Expected: FAIL because no CAS store exists.

- [ ] **Step 3: Implement isolated coordination commits.**

Fetch only the configured coordination ref. Read its tree without checkout. Create the next tree in an isolated temporary repository or index, commit normalized live state, then push with an explicit lease against the expected remote object ID. Re-fetch and compare the final content and revision.

- [ ] **Step 4: Make failure classifications exact.**

Return typed errors for no remote, authentication unavailable, expected revision mismatch, non-fast coordination history, malformed state, push rejected, and postcondition mismatch. No remote produces single-user mode, not a collaborative claim.

- [ ] **Step 5: Replace branch-scanning claims.**

Remove `ReadRemoteFiles` from claim truth after the new store tests pass. Feature branches may contain work evidence, but cannot own live coordination status.

- [ ] **Step 6: Run race and E2E tests, then commit.**

```sh
cd cli && go test -race ./internal/provider ./internal/gitx ./internal/command -run 'GitLocal|Claim' -v
cd ..
git add cli/internal/provider cli/internal/gitx cli/internal/command
git commit -m "feat(work): coordinate Git-local claims safely"
```

### Task 7: Add safe branch, worktree, and submodule mutations with postconditions

**Files:**
- Create: `cli/internal/gitx/mutate.go`
- Create: `cli/internal/gitx/mutate_test.go`
- Modify: `cli/internal/gitx/worktree.go`
- Modify: `cli/internal/gitx/submodule.go`
- Modify: `cli/internal/operation/plan.go`
- Create: `cli/internal/operation/command.go`
- Create: `cli/internal/operation/command_test.go`
- Modify: `cli/internal/command/git.go`
- Create: `cli/internal/command/work_lifecycle.go`

**Interfaces:**
- Produces `gitx.CreateWorktree(ctx, request) domain.Result` with the verified worktree path, branch, and HEAD in `Facts`.
- Produces `gitx.SyncPinnedSubmodules(ctx, root, paths) domain.Result`.
- Produces a restricted command executor that accepts only preconstructed Git operations and verifies postconditions.

- [ ] **Step 1: Write failing safety tests.**

```go
func TestCreateWorktreeRefusesDirtyBaseAndDuplicateBranch(t *testing.T) {
    repo := newRepository(t)
    makeDirty(t, repo)
    result := gitx.CreateWorktree(context.Background(), gitx.CreateWorktreeRequest{
        Root: repo, Branch: "feature/account-recovery", Base: "main",
    })
    assert.Equal(t, domain.StatusBlocked, result.Status)
    assert.Contains(t, codes(result.Blockers), "git.base-dirty")
}

func TestSyncPinnedSubmoduleVerifiesExactPostcondition(t *testing.T) {
    fixture := newRootWithChildSubmodule(t)
    deinitializeChild(t, fixture)
    result := gitx.SyncPinnedSubmodules(context.Background(), fixture.Root, []string{"backend"})
    assert.Equal(t, domain.StatusPassed, result.Status)
    assert.Equal(t, fixture.ExpectedBackend, inspectHead(t, fixture.Child))
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/gitx ./internal/operation ./internal/command -run 'CreateWorktree|SyncPinned|Command' -v`

Expected: FAIL because commands are plan-only.

- [ ] **Step 3: Implement an allow-listed Git executor.**

Do not make `operation.Apply` execute arbitrary `CommandStep`. Add typed operations for `git worktree add` and `git submodule update --init -- <path>`; reuse the restricted coordination push path completed in Task 6. Validate executable identity, exact arguments, directory, preconditions, timeout, output limit, and postcondition.

- [ ] **Step 4: Implement safe worktree creation.**

Reject invalid or AI-marked branches, dirty/diverged base, missing base, branch already checked out, target inside a repository, symlinked target ancestors, and existing target. Verify returned worktree path, branch, and HEAD after creation; remove only an empty failed target created by this operation.

- [ ] **Step 5: Implement pinned submodule initialization.**

Initialize only explicit safe paths from the root manifest. Reject dirty, unsafe URL, existing pointer mismatch, recursive implicit sync, and a root identity mismatch. Verify actual child HEAD equals the root gitlink afterwards.

- [ ] **Step 6: Replace plan-only command UX.**

Keep plan output for diagnostics, add Skill-facing apply commands, and never expose arbitrary command execution. A detached submodule is informational unless development starts.

- [ ] **Step 7: Run tests and commit.**

```sh
cd cli && go test -race ./internal/gitx ./internal/operation ./internal/command -v
cd ..
git add cli/internal/gitx cli/internal/operation cli/internal/command
git commit -m "feat(git): apply isolated workspace operations"
```

### Task 8: Complete lifecycle transitions and commit-bound evidence

**Files:**
- Create: `cli/internal/evidence/model.go`
- Create: `cli/internal/evidence/record.go`
- Create: `cli/internal/evidence/verify.go`
- Create: `cli/internal/evidence/evidence_test.go`
- Create: `schemas/evidence.schema.json`
- Copy: `cli/internal/schema/definitions/evidence.schema.json`
- Create: `cli/internal/work/lifecycle.go`
- Create: `cli/internal/work/lifecycle_test.go`
- Modify: `cli/internal/command/work_lifecycle.go`
- Delete after replacement: `cli/internal/project/work.go`

**Interfaces:**
- Produces `evidence.Record`, `evidence.Run(ctx, Request) (Record, domain.Result)`, and `evidence.VerifyCurrent(record, actual) []domain.Item`.
- Produces `work.Transition(definition Definition, live LiveState, records []evidence.Record, target State) domain.Result`; `work.LiveState` is populated by the command adapter and does not import `provider`.

- [ ] **Step 1: Write failing evidence and transition tests.**

```go
func TestEvidenceBecomesStaleWhenWorkspaceHeadChanges(t *testing.T) {
    repo := newRepository(t)
    record := runPassingEvidence(t, repo, "go test ./...")
    commitFile(t, repo, "after.txt", "changed")
    issues := evidence.VerifyCurrent(record, evidence.Actual{Workspace: repo, Head: head(t, repo)})
    assert.Contains(t, codes(issues), "evidence.commit-changed")
}

func TestParentCannotBecomeDoneBeforeChildrenIntegrated(t *testing.T) {
    live := work.LiveState{Status: "in_progress", Owner: "owner-a", Revision: "42", Confirmed: true}
    result := work.Transition(parentDefinition(), live, evidenceSet(), work.Done)
    assert.Equal(t, domain.StatusBlocked, result.Status)
    assert.Contains(t, codes(result.Blockers), "work.children-not-integrated")
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/evidence ./internal/work ./internal/command -run 'Evidence|Transition|Finish' -v`

Expected: FAIL because current finish accepts arbitrary strings and has no integrated state.

- [ ] **Step 3: Define evidence records.**

```go
type Record struct {
    SchemaVersion int       `json:"schema_version" yaml:"schema_version"`
    ID            string    `json:"id" yaml:"id"`
    Kind          string    `json:"kind" yaml:"kind"`
    WorkID        string    `json:"work_id" yaml:"work_id"`
    WorkspaceID   string    `json:"workspace_id" yaml:"workspace_id"`
    Command       []string  `json:"command,omitempty" yaml:"command,omitempty"`
    StartedAt     time.Time `json:"started_at" yaml:"started_at"`
    FinishedAt    time.Time `json:"finished_at" yaml:"finished_at"`
    ExitCode      int       `json:"exit_code" yaml:"exit_code"`
    Commit        string    `json:"commit" yaml:"commit"`
    DefinitionFingerprint string `json:"definition_fingerprint" yaml:"definition_fingerprint"`
    ContractFingerprint string `json:"contract_fingerprint" yaml:"contract_fingerprint"`
    OutputDigest  string    `json:"output_digest" yaml:"output_digest"`
    ArtifactDigests map[string]string `json:"artifact_digests,omitempty" yaml:"artifact_digests,omitempty"`
}
```

Define the lifecycle input in `cli/internal/work/model.go`:

```go
type LiveState struct {
    Status    string
    Owner     string
    Revision  string
    Confirmed bool
}
```

- [ ] **Step 4: Implement restricted evidence execution.**

Commands come from the workspace’s approved command configuration, not issue free text. Capture bounded sanitized output, exact timing, exit status, current clean commit, and fingerprints. A dirty workspace cannot produce reusable release evidence.

- [ ] **Step 5: Implement lifecycle invariants.**

Require current provider owner/revision for `in_progress`, current implementation evidence for `review`, child merge evidence for `integrated`, and all child/integration/root-pointer/user requirements for `done`. `handoff` is allowed only when ownership changes and records exact branch, commit, local-only risk, evidence, blocker, and next action.

- [ ] **Step 6: Remove string-only finish and migrate commands.**

`work finish` becomes a verified transition alias; `work transition` exposes the complete machine surface. Close or release the live claim only after provider transition and postcondition re-read succeed.

- [ ] **Step 7: Run tests and commit.**

```sh
cd cli && go test -race ./internal/evidence ./internal/work ./internal/provider ./internal/command -v
cd ..
git add cli/internal/evidence cli/internal/work cli/internal/provider cli/internal/command cli/internal/schema/definitions schemas
git commit -m "feat(work): verify lifecycle evidence"
```

---

## Milestone 3 — Service meaning, integration, and release identity

### Task 9: Expand contracts into a service-obligation graph

**Files:**
- Modify: `cli/internal/contract/check.go`
- Modify: `cli/internal/contract/compatibility.go`
- Create: `cli/internal/contract/registry.go`
- Create: `cli/internal/contract/registry_test.go`
- Replace: `schemas/contract-registry.schema.json`
- Replace: `cli/internal/schema/definitions/contract-registry.schema.json`
- Modify: `cli/internal/context/index.go`
- Modify: `cli/internal/context/graph.go`
- Modify: `cli/internal/policy/conflict.go`
- Modify: `templates/project/contracts/registry.yaml`

**Interfaces:**
- Produces typed contract kinds `product`, `business`, `behavior`, `interface`, and `data`.
- Produces `contract.LoadRegistry(root) (Registry, error)` and `contract.Impact(registry, id) Impact`.

- [ ] **Step 1: Write failing obligation and impact tests.**

```go
func TestBusinessContractRequiresObservableRejectedAndFailureBehavior(t *testing.T) {
    definition := contract.Definition{ID: "contract.business.account-recovery", Kind: "business"}
    issues := contract.Check(definition)
    assert.Contains(t, codes(issues), "contract.rejection-behavior-required")
    assert.Contains(t, codes(issues), "contract.failure-behavior-required")
}

func TestContractImpactIncludesUIDataMigrationAndActiveWork(t *testing.T) {
    impact := contract.Impact(fixtureRegistry(t), "contract.behavior.refund-timeout")
    assert.ElementsMatch(t, []string{"ui.refund", "data.refund", "migration.refund", "work.refund-ui"}, impact.Dependents)
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/contract ./internal/context ./internal/policy -run Contract -v`

Expected: FAIL because the current contract definition only covers a narrower behavior shape.

- [ ] **Step 3: Define registry entries and obligations.**

Each entry contains kind, source, status, providers, consumers, related product/scenario/data/UI IDs, compatibility mode, and fingerprint. Kind-specific documents define purpose/non-goals, eligibility/invariants, observable outcomes, interfaces, or data lifecycle.

- [ ] **Step 4: Build deterministic impact edges.**

Index registry and document refs once. Mark dependent UI, interface, DBML, migration, tests, and work definitions stale when the contract fingerprint changes.

- [ ] **Step 5: Upgrade conflict severity.**

The same business, behavior, or migration contract is `block`; overlapping compatible consumer work is `coordinate`; missing or stale registry evidence is `unknown`.

- [ ] **Step 6: Run tests and commit.**

```sh
cd cli && go test -race ./internal/contract ./internal/context ./internal/policy ./internal/command -v
cd ..
git add cli/internal/contract cli/internal/context cli/internal/policy schemas templates
git commit -m "feat(contract): model service obligations"
```

### Task 10: Complete DBML/dbdiagram and external UI reconciliation

**Files:**
- Modify: `cli/internal/database/dbdiagram.go`
- Modify: `cli/internal/database/dbdiagram_test.go`
- Create: `cli/internal/database/reconcile.go`
- Create: `cli/internal/database/reconcile_test.go`
- Modify: `cli/internal/ui/import.go`
- Modify: `cli/internal/ui/import_test.go`
- Create: `cli/internal/ui/reconcile.go`
- Create: `cli/internal/ui/reconcile_test.go`
- Modify: `cli/internal/command/boundaries.go`
- Create: `schemas/external-source.schema.json`
- Copy: `cli/internal/schema/definitions/external-source.schema.json`

**Interfaces:**
- Produces `database.PrepareProposal`, `database.ReconcileProposal`, and migration impact.
- Produces `ui.Register`, `ui.Reconcile`, and authority-aware stale results.
- Adds `db diagram prepare|reconcile` and `ui import|reconcile` command paths.

- [ ] **Step 1: Write failing reconciliation tests.**

```go
func TestDBDiagramPullCannotOverwriteCanonicalDBML(t *testing.T) {
    root := dbFixture(t)
    proposal := database.PrepareProposal(root, candidateDBML())
    assert.FileExists(t, proposal.Path)
    assert.Equal(t, canonicalDBML(), readCanonical(t, root))
    assert.Contains(t, proposal.Diff.ChangedColumns, "accounts.recovery_state")
}

func TestCanonicalUIChangeMarksMappedFlowsStale(t *testing.T) {
    state := ui.Reconcile(canonicalSource(), changedArchive())
    assert.Contains(t, state.StaleRefs, "ui.account-recovery")
    assert.True(t, state.RequiresApproval)
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/database ./internal/ui ./internal/command -run 'Reconcile|Proposal|Authority' -v`

Expected: FAIL because current code stops at isolated preparation/import.

- [ ] **Step 3: Implement dbdiagram proposal provenance.**

Record canonical fingerprint, proposal hash, official CLI identity/version, project ID, action, fetch time, and semantic diff under `.harness/local/`. The Skill runs the selected official CLI explicitly; the CLI never prints or persists the token.

- [ ] **Step 4: Implement DB reconciliation.**

Reject stale base fingerprints and malformed DBML. Return contract, entity, migration-order, test, and rollback impact. Apply to canonical Git DBML only through an atomic reviewed plan.

- [ ] **Step 5: Implement authority-aware UI reconciliation.**

Keep ZIP traversal, symlink, bomb, license, and size protections. `reference` has no automatic authority, `seed` can evolve after initial import, and `canonical` changes mark mapped UI flows and consumers stale.

- [ ] **Step 6: Run tests and commit.**

```sh
cd cli && go test -race ./internal/database ./internal/ui ./internal/context ./internal/command -v
cd ..
git add cli/internal/database cli/internal/ui cli/internal/command cli/internal/schema/definitions schemas
git commit -m "feat(boundaries): reconcile database and UI sources"
```

### Task 11: Drive integration and release from actual workspace state

**Files:**
- Create: `cli/internal/integration/model.go`
- Create: `cli/internal/integration/plan.go`
- Create: `cli/internal/integration/verify.go`
- Create: `cli/internal/integration/integration_test.go`
- Modify: `cli/internal/command/integrate.go`
- Create: `cli/internal/release/collect.go`
- Create: `cli/internal/release/collect_test.go`
- Modify: `cli/internal/release/candidate.go`
- Modify: `cli/internal/release/gate.go`
- Modify: `cli/internal/command/release.go`
- Replace: `schemas/release-candidate.schema.json`
- Replace: `schemas/release-validation.schema.json`

**Interfaces:**
- Produces `integration.Plan(definitions, providerStates, workspaceStates) Plan` and `integration.Verify(plan, evidence) domain.Result`.
- Produces `release.CollectInput(ctx, root, evidenceStore) (Input, []domain.Item)`.
- Removes the requirement for users to hand-author release input JSON in normal use.

- [ ] **Step 1: Write failing exact-state tests.**

```go
func TestIntegrationRequiresProviderConsumerAndRootPointerOrder(t *testing.T) {
    plan := integration.Plan(featureDefinitions(), providerStates(), workspaceStates())
    assert.Equal(t, []string{"contract", "backend", "frontend", "root-pointer"}, plan.Order)
}

func TestCollectedReleaseInputRejectsPointerMismatchAndStaleEvidence(t *testing.T) {
    fixture := releaseFixture(t)
    fixture.Backend.SetPointerMismatch()
    fixture.Evidence.AdvanceCommit()
    _, issues := release.CollectInput(context.Background(), fixture.Root, fixture.Evidence)
    assert.Contains(t, codes(issues), "release.pointer-mismatch")
    assert.Contains(t, codes(issues), "release.evidence-stale")
}
```

- [ ] **Step 2: Run and verify failure.**

Run: `cd cli && go test ./internal/integration ./internal/release ./internal/command -run 'Integration|CollectInput|Release' -v`

Expected: FAIL because integration is fixed text and release input is user-authored.

- [ ] **Step 3: Implement work-driven integration planning.**

Topologically order shared contract, providers, consumers/mocks, UI connection, migration/rollback, and root pointer. Detect cycles, missing child merge evidence, stale provider state, overlapping pointer ownership, and compatibility-mode violations.

- [ ] **Step 4: Verify exact integration.**

Bind every step to work-definition, contract, provider revision, workspace remote, merge commit, and evidence. A child PR can reach `integrated`; the parent reaches `done` only after root pointers and cross-repository tests match.

- [ ] **Step 5: Collect release inputs from actual state.**

Read root HEAD and gitlinks, child repository remotes/HEADs, canonical fingerprints, artifact digests, current evidence, profile, and tool versions. Block dirty, diverged, mismatched, unknown-provider, and stale-evidence state before creating a candidate.

- [ ] **Step 6: Preserve same-candidate user validation.**

Technical collection and checks create the digest first. User validation references that digest. Any root, child, contract, data, artifact, evidence, profile, or tool identity change invalidates it.

- [ ] **Step 7: Run tests and commit.**

```sh
cd cli && go test -race ./internal/integration ./internal/release ./internal/evidence ./internal/workspace ./internal/command -v
cd ..
git add cli/internal/integration cli/internal/release cli/internal/command schemas testdata/releases
git commit -m "feat(release): bind exact service integration"
```

---

## Milestone 4 — Product UX, dogfooding, and public readiness

### Task 12: Rewrite five Skills around the completed deterministic surface

**Files:**
- Modify: `skills/start-project/SKILL.md`
- Modify: `skills/continue-project/SKILL.md`
- Modify: `skills/plan-project-work/SKILL.md`
- Modify: `skills/coordinate-project-work/SKILL.md`
- Modify: `skills/recover-and-release-project/SKILL.md`
- Modify: `references/workflow.md`
- Modify: `references/safety.md`
- Modify: `references/context-recovery.md`
- Modify: `templates/project/.agents/skills/use-project-harness/SKILL.md`
- Modify: `templates/project/.agents/skills/use-project-harness/references/fallback.md`
- Replace: `testdata/plugin/behavior.json`
- Create: `evals/agent-behavior/scenarios.yaml`
- Create: `evals/agent-behavior/rubric.yaml`
- Create: `scripts/validate_agent_eval.py`
- Create: `scripts/validate_agent_eval_test.py`
- Create: `scripts/run_agent_eval.py`

**Interfaces:**
- Skills start with `orchestrator status --json`, not old document-only `context audit`.
- Skills use connector snapshots and deterministic CLI results, hide internal mechanics, and ask one material question at a time.
- Evaluation scenarios cover routing and behavior, not only string presence.

- [ ] **Step 1: Use `superpowers:writing-skills` and write failing behavior scenarios first.**

```yaml
- id: continue-after-clean-clone
  prompt: "이 프로젝트 이어서 해."
  expected_skill: continue-project
  required_actions:
    - combined_status_before_mutation
    - report_confirmed_stale_unknown
    - one_safe_next_action
  forbidden_actions:
    - repeat_answered_question
    - claim_cached_provider_as_live
    - expose_operation_id

- id: start-cross-repo-conflict
  prompt: "환불 실패 UI 작업 시작해줘."
  expected_skill: coordinate-project-work
  required_actions:
    - provider_reread
    - semantic_conflict_preflight
    - contract_first_resolution
  forbidden_actions:
    - create_branch_before_claim
    - ai_marker_in_git_name
```

- [ ] **Step 2: Run validators and observe failure.**

Run:

```sh
python3 scripts/validate_plugin_test.py -v
python3 scripts/validate_agent_eval_test.py -v
python3 scripts/validate_agent_eval.py .
```

Expected: FAIL because Skills reference old commands and behavior eval files do not exist.

- [ ] **Step 3: Rewrite each Skill with one responsibility.**

All Skills perform status recovery before mutation, locate canonical refs, distinguish fact/assumption/open/stale, use external connectors only when selected, and translate JSON into plain language. Discovery asks a single material question and checkpoints immediately. Work Skills require provider re-read and post-write re-read.

- [ ] **Step 4: Add just-in-time tool selection.**

The workflow derives requirements, detects installed Plugin/Skill/CLI/MCP, searches current official sources when available, compares two or three candidates on capability/maintenance/security/license/platform/export/lock-in/cost, records a dated decision, and connects only the selection. No static tool list is called universally current.

- [ ] **Step 5: Add honest connector routing.**

Git-local uses the CLI. GitHub uses an installed GitHub connector or authenticated `gh` only when selected. Jira uses an installed Atlassian connector only when selected. Beads uses an installed `bd --json`. Missing connectors produce choices or reduced Git-local mode; they never fabricate external state.

- [ ] **Step 6: Run Skill drill evaluations.**

Use the official Skill validator and a real Codex drill through `scripts/run_agent_eval.py`. The runner invokes the installed `codex` command in read-only or workspace-write fixtures according to each scenario, writes transcripts only under `.harness/local/evals/`, and scores them with the checked-in rubric. Absence of a working Codex command is a local release blocker, not a skipped pass. Capture transcripts for: new project, clean-clone resume, forgotten context, semantic conflict, provider unavailable, local-only work, and release mismatch. The rubric fails repeated questions, unsafe mutation, false claim, internal jargon, and unsupported provider claims.

Run:

```sh
python3 scripts/run_agent_eval.py \
  --command codex \
  --scenarios evals/agent-behavior/scenarios.yaml \
  --rubric evals/agent-behavior/rubric.yaml \
  --output .harness/local/evals
```

Expected: all required scenarios produce transcripts and a passing rubric report; no transcript is committed.

- [ ] **Step 7: Commit.**

```sh
git add skills references templates/project/.agents testdata/plugin evals scripts/validate_agent_eval*
git commit -m "feat(plugin): guide verified service continuity"
```

### Task 13: Package and bootstrap the cross-platform CLI honestly

**Files:**
- Create: `scripts/bootstrap-cli.sh`
- Create: `scripts/bootstrap-cli.ps1`
- Create: `scripts/bootstrap_cli_test.py`
- Create: `scripts/render_plugin_packages.py`
- Create: `scripts/render_plugin_packages_test.py`
- Modify: `.goreleaser.yaml`
- Modify: `.github/workflows/release.yml`
- Modify: `.codex-plugin/plugin.json`
- Modify: `.agents/plugins/marketplace.json`
- Modify: `skills/start-project/SKILL.md`
- Modify: `skills/continue-project/SKILL.md`
- Modify: `hooks/hooks.json`

**Interfaces:**
- Local development honors `ORCHESTRATOR_CLI` and repository-built binaries.
- Release packages provide checksummed darwin/amd64, darwin/arm64, windows/amd64, and windows/arm64 CLI assets.
- First explicit product use may offer the matching verified install; SessionStart never downloads software.

- [ ] **Step 1: Write failing bootstrap and package tests.**

```python
def test_bootstrap_selects_exact_platform_asset_and_verifies_checksum(self):
    result = run_bootstrap(os_name="windows", arch="arm64", fixture_release=FIXTURE)
    self.assertEqual("orchestrator_windows_arm64.exe", result.asset)
    self.assertTrue(result.checksum_verified)

def test_session_hook_never_downloads(self):
    hooks = (ROOT / "hooks" / "hooks.json").read_text()
    self.assertNotIn("curl", hooks)
    self.assertNotIn("Invoke-WebRequest", hooks)
```

- [ ] **Step 2: Run and verify failure.**

Run: `python3 scripts/bootstrap_cli_test.py -v && python3 scripts/render_plugin_packages_test.py -v`

Expected: FAIL because bootstrap and platform packages do not exist.

- [ ] **Step 3: Implement checksum-first bootstrap scripts.**

They accept an explicit release base URL and version, choose only a supported exact asset, download to a temporary path, verify SHA-256 from a pinned checksum manifest, set executable permissions where applicable, atomically install into the user-selected tool directory, and run `orchestrator doctor --json`. They never execute unverified bytes or edit shell profiles.

- [ ] **Step 4: Keep hooks fail-safe.**

Hooks call a resolved `orchestrator` only after installation. Before installation they exit without mutation; the repo-local Skill performs reduced preflight. The first explicit natural-language product request explains and performs the supported install path according to user and environment policy.

- [ ] **Step 5: Render platform release packages.**

GoReleaser produces CLI archives and checksums. The packaging script combines the common Plugin files with platform guidance and validates that every package references the same Plugin and CLI version. The public base URL remains configuration until the repository decision is supplied.

- [ ] **Step 6: Cross-build and test.**

Run:

```sh
python3 scripts/bootstrap_cli_test.py -v
python3 scripts/render_plugin_packages_test.py -v
cd cli
for target in darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  os=${target%/*}; arch=${target#*/}; ext=""; [ "$os" = windows ] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -o "../dist/orchestrator_${os}_${arch}${ext}" ./cmd/orchestrator
done
```

Expected: four binaries and passing package tests.

- [ ] **Step 7: Commit.**

```sh
git add scripts .goreleaser.yaml .github/workflows/release.yml .codex-plugin .agents/plugins skills hooks
git commit -m "feat(distribution): bootstrap verified CLI builds"
```

### Task 14: Dogfood an actual multi-repository service and measure the claim

**Files:**
- Create: `dogfood/README.md`
- Create: `dogfood/scenario.yaml`
- Create: `dogfood/run.sh`
- Create: `dogfood/run.ps1`
- Create: `dogfood/expected-results.json`
- Create: `dogfood/report.md`
- Create: `cli/internal/command/production_e2e_test.go`
- Create: `evals/baseline/scenarios.yaml`
- Create: `evals/baseline/score.py`
- Create: `evals/baseline/score_test.py`

**Interfaces:**
- Builds temporary orchestration, frontend, and backend repositories plus bare remotes and actual submodules.
- Executes the installed CLI and repo-local Skill flow without modifying product source.
- Produces reproducible machine results and a human-readable comparison report.

- [ ] **Step 1: Write the failing production scenario contract.**

The scenario must create:

```text
orchestration root
├── business and failure contracts
├── account-recovery parent work
├── frontend submodule
└── backend submodule
```

It must simulate two owners, provider/consumer work, a path-disjoint shared behavior conflict, a Git-local claim race, contract-first resolution, frontend mock and backend provider evidence, child merges, pointer integration, clean clone, local-only warning, context recovery, and exact RC verification.

- [ ] **Step 2: Run the empty scenario and verify failure.**

Run: `bash dogfood/run.sh`

Expected: FAIL until all commands and expected result codes exist.

- [ ] **Step 3: Implement POSIX and PowerShell runners using temporary directories.**

Both runners use ordinary Git commands only to construct fixtures; all product behavior is invoked through `orchestrator`. They compare stable JSON codes and do not depend on a public provider account.

- [ ] **Step 4: Run the product on itself.**

Adopt this repository with the completed Skills and CLI, generate a combined status, define the remaining documentation/release work, exercise Git-local coordination on a temporary remote, and capture privacy-safe evidence in `dogfood/report.md`. Do not alter the product’s main history through the dogfood fixture.

- [ ] **Step 5: Add baseline metrics.**

Compare manual Git + static docs against the harness for clean-clone next-action time, repeated material questions, false live claims, missed semantic conflicts, wrong-workspace edits, pointer drift, local-only misclassification, and invalid RC combinations. Report raw scenario counts and limitations; do not invent performance numbers.

- [ ] **Step 6: Run production E2E and score tests.**

Run:

```sh
bash dogfood/run.sh
cd cli && go test -race ./internal/command -run ProductionE2E -v
cd .. && python3 evals/baseline/score_test.py -v
```

Expected: PASS and a report whose values are derived from captured machine results.

- [ ] **Step 7: Commit.**

```sh
git add dogfood evals/baseline cli/internal/command/production_e2e_test.go
git commit -m "test(product): prove multi-repository continuity"
```

### Task 15: Finish bilingual UX, security, CI, and local release readiness

**Files:**
- Replace: `README.md`
- Replace: `README.ko.md`
- Modify: `docs/getting-started/en.md`
- Modify: `docs/getting-started/ko.md`
- Modify: `docs/concepts/en.md`
- Modify: `docs/concepts/ko.md`
- Modify: `docs/guides/{new-project,existing-project,submodules,dbdiagram,release,troubleshooting}-{en,ko}.md`
- Modify: `docs/security/privacy-{en,ko}.md`
- Modify: `docs/security/threat-model-{en,ko}.md`
- Replace: `docs/release-readiness.md`
- Modify: `scripts/validate_docs.py`
- Modify: `scripts/security_scan.py`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/security.yml`
- Modify: `profiles/strict-release/**`
- Delete: `docs/superpowers/specs/2026-07-17-product-critical-review-working-notes.md`

**Interfaces:**
- README leads with natural-language use, the differentiator, a five-minute path, and truthful external-tool boundaries.
- Documentation explains generated files, collaboration, recovery, provider choices, core/strict release, installation, and troubleshooting in Korean and English parity.
- CI proves native macOS/Windows behavior where hosted runners are available and cross-builds every supported target.

- [ ] **Step 1: Strengthen documentation and security tests before rewriting docs.**

Add assertions for:

```text
both languages describe the same five Skills
both languages describe combined status and provider truth
both languages distinguish Memory from repository evidence
both languages show no AI markers in Git names
README does not claim universal providers or automatic production
all embedded commands exist in CLI help
all generated paths match the actual project fixture
```

- [ ] **Step 2: Run validators and verify failure against old docs.**

Run: `python3 scripts/validate_docs.py && python3 scripts/security_scan.py .`

Expected: FAIL on old command names, old generated layout, or missing provider/recovery descriptions.

- [ ] **Step 3: Rewrite bilingual docs around actual user journeys.**

Cover start/adopt, continue, plan/claim, contract/DB/UI/integration, and recover/release. Include external tool detection and selection, Git-local/GitHub/Jira/Beads capability differences, root-child bridge, dirty/diverged recovery, plugin-less behavior, and strict-release differences. Keep CLI reference secondary.

- [ ] **Step 4: Expand the threat model.**

Cover provider and Memory prompt injection, malicious archives/DBML, coordination-branch race, unsafe submodule URLs, symlinks, command allow-listing, output limits, stale snapshots, local-only false recovery, bootstrap checksums, diagnostic redaction, and exact-RC tampering.

- [ ] **Step 5: Upgrade CI and validation.**

Run unit/integration tests, race on supported native runners, fuzz smoke, static analysis, vulnerability scan, secret/security scan, official Plugin/Skill validation, hook behavior, plugin-less E2E, dogfood, docs parity, strict profile, snapshot packaging, and macOS/Windows cross-builds. Hosted external provider writes remain an explicitly named manual release check.

- [ ] **Step 6: Remove temporary review notes after coverage audit.**

Verify every retained requirement from the working note is represented in the final design, plan, code, or user docs. Delete only the temporary working note created for the long review; keep the final design and implementation plan.

- [ ] **Step 7: Run the full local release matrix.**

Run:

```sh
cd cli
go test ./...
go test -race ./...
go vet ./...
go test ./internal/context -run '^$' -fuzz FuzzFingerprint -fuzztime 15s
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
cd ..
python3 scripts/validate_plugin_test.py -v
python3 scripts/validate_plugin.py .
python3 scripts/validate_agent_eval_test.py -v
python3 scripts/validate_agent_eval.py .
python3 scripts/validate_docs.py
python3 scripts/security_scan.py .
python3 scripts/validate_release_config.py .
python3 -m unittest discover -s profiles/strict-release/scripts -p '*_test.py' -v
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/*.yml
go run github.com/goreleaser/goreleaser/v2@latest check
bash dogfood/run.sh
git diff --check
```

Expected: every command exits zero. Any unavailable network or hosted-account check is recorded as unverified rather than passed.

- [ ] **Step 8: Perform a requirement-by-requirement completion audit.**

For every item in Design §21, record authoritative evidence: test name/output, dogfood step, generated artifact, rendered doc, or external blocker. A passing umbrella script is insufficient without scenario coverage.

- [ ] **Step 9: Commit local release readiness.**

```sh
git add README.md README.ko.md docs scripts .github profiles
git commit -m "docs: complete product release guidance"
```

## Final external blockers

Implementation and local release preparation do not infer:

- public product name and branding;
- public GitHub organization/repository and release base URL;
- publisher or marketplace account;
- code-signing and package-signing identities;
- real Jira tenant and production GitHub/Jira write approval;
- irreversible Plugin marketplace, package-manager, or release publication.

Until real GitHub/Jira write drills are completed, documentation describes those adapters as available through selected connectors with locally verified protocol behavior, not as hosted end-to-end certified.

## Self-review checklist

| Design requirement | Implementation and evidence tasks |
|---|---|
| §§4–6 product boundary, UX, five Skills | Tasks 1, 12, 13, 15 |
| §7 repository and root-child architecture | Task 2 and Task 14 real-submodule fixture |
| §8 source hierarchy and recovery | Tasks 2, 3, 5, 8, 12, 14 |
| §9 discovery and coverage | Existing checkpoint core, Task 12 behavior eval, Task 14 dogfood |
| §10 service contracts | Task 9 and Task 14 contract-first scenario |
| §11 work/provider lifecycle | Tasks 4–8 and Task 14 claim race |
| §§12–13 Git and semantic conflicts | Tasks 6, 7, 9, 14 |
| §14 DBML/dbdiagram and UI | Task 10 and production E2E in Task 14 |
| §15 technology/tool selection | Task 12 official-source decision behavior |
| §16 evidence and release | Tasks 8, 11, 14 |
| §17 security/privacy | Tasks 1, 2, 5–15 and the final threat-model matrix |
| §18 deterministic CLI | Tasks 3–11 |
| §19 distribution/portability | Tasks 1, 2, 12, 13, 15 |
| §20 current-code migration | Every replacement task; deletions occur only after passing replacements |
| §§21–22 acceptance and failure behavior | Tasks 14–15 requirement audit |

- [x] Every requirement in Design §§4–22 maps to at least one task and an authoritative acceptance check.
- [x] No task relies on the deleted broad provider architecture or default strict publication gates.
- [x] Git-local collaboration is a remote compare-and-swap protocol, not a local advisory file.
- [x] Provider snapshots remain local and untrusted; live state remains in exactly one provider.
- [x] Generated context caches are never canonical.
- [x] Child PR completion, parent integration, and exact root pointer completion are distinct.
- [x] Evidence binds to commits and fingerprints rather than user-provided strings.
- [x] Plugin Hook, CLI bootstrap, repo-local Skill, and Markdown fallback each have a failure test.
- [x] Dogfood uses real Git repositories, remotes, submodules, worktrees, and race scenarios.
- [x] Public claims are limited to the evidence actually produced.
