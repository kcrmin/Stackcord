# Full-stack Orchestrator Production Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and publish a local-first Codex Plugin plus cross-platform Go CLI that can create or adopt a framework-neutral full-stack project, preserve its context across people and AI sessions, coordinate Git/submodule work, and verify the same release candidate through production release.

**Architecture:** A deterministic Go CLI owns schemas, actual-state discovery, policy, plan/apply operations, adapters, and release gates. Thin Agent Skills route natural-language intent to that CLI; generated repositories retain a small repo-local Skill and versioned harness files so clone continuation does not depend on the Plugin. External providers are capability-negotiated adapters and never replace canonical Git specifications or contracts.

**Tech Stack:** Go 1.26.5, Cobra v1.10.2, `go.yaml.in/yaml/v3` v3.0.4, `github.com/santhosh-tekuri/jsonschema/v6` v6.0.2, Testify v1.11.1, JSON Schema Draft 2020-12, Go standard `os/exec` Git adapter, Agent Skills, Codex Plugin manifest, GitHub Actions, GoReleaser v2.17.0, Cosign/Sigstore, Syft SBOM, Homebrew, WinGet.

## Global Constraints

- The public first release is the complete production `1.0.0` product defined by the design gates.
- macOS arm64/x86_64 and Windows arm64/x86_64 are first-class targets.
- Core operation must not require Node.js, a daemon, a central server, or telemetry.
- Generated projects remain framework, language, database, cloud, and Git-host neutral.
- Product meaning lives in `specs/`, obligations in `contracts/`, control state in `.harness/`, and guides/runbooks in `docs/`.
- Behavior, bug, contract, migration, security, UI interaction, Git mutation, and lifecycle changes use test-first development.
- `main` is protected; use short-lived branches, Conventional Commits, squash merge by default, and no AI markers in branch/commit names.
- No hidden pull, rebase, stash, reset, clean, force-push, external write, package install, or production action.
- Git DBML is canonical; dbdiagram pull always lands in an isolated scratch directory before semantic review.
- English is canonical and Korean must have section-level semantic parity.
- The working binary name is `orchestrator`; perform the approved public-name rename once before the first public package is created.
- Use Go 1.26.5 security patch level for implementation; YAML v4 is still a release candidate on 2026-07-16, so use stable v3.0.4 and re-evaluate v4 only after a stable release and migration tests.

---

## File Map

```text
.codex-plugin/plugin.json                 Codex package manifest
.agents/plugins/marketplace.json          GitHub-installable marketplace entry
skills/*/SKILL.md                         Intent-specific AI workflows
hooks/hooks.json                          Optional SessionStart/PostCompact reminders
references/*.md                           Shared Skill reference material
cli/cmd/orchestrator/main.go              Binary entry point
cli/internal/domain/*.go                  Stable domain types and invariants
cli/internal/schema/*.go                  YAML/JSON loading and validation
cli/internal/context/*.go                 Fingerprint, index, graph, stale propagation
cli/internal/operation/*.go               Plan/apply journal and recovery
cli/internal/policy/*.go                  Approval and conflict policy
cli/internal/gitx/*.go                    Safe Git/submodule/worktree adapter
cli/internal/provider/*.go                Capability-negotiated external adapters
cli/internal/project/*.go                 Init/adopt/generation lifecycle
cli/internal/release/*.go                 RC and release verification
schemas/*.json                            Harness and result JSON Schemas
templates/project/**                      Framework-neutral generated project files
locales/en/*.json                         Canonical CLI text
locales/ko/*.json                         Korean parity text
testdata/**                               Real Git topology and malicious input fixtures
docs/**                                   User, operator, security, contributor material
packaging/**                              Homebrew and WinGet metadata
```

## Task 1: Establish the Go module, result envelope, and command shell

**Files:**
- Create: `cli/go.mod`
- Create: `cli/cmd/orchestrator/main.go`
- Create: `cli/internal/domain/result.go`
- Create: `cli/internal/output/json.go`
- Create: `cli/internal/output/json_test.go`
- Create: `schemas/result.schema.json`

**Interfaces:**
- Produces: `domain.Result`, `domain.Status`, `output.WriteJSON(io.Writer, domain.Result) error`, and the stable exit-code mapping used by every later task.

- [ ] **Step 1: Write the failing JSON-envelope test**

```go
func TestWriteJSONUsesStableEnvelope(t *testing.T) {
    result := domain.Result{
        SchemaVersion: "1.0", ToolVersion: "1.0.0", Command: "doctor",
        OperationID: "01JTEST", Status: domain.StatusPassed, ExitCode: 0,
        Summary: "Environment is ready.",
    }
    var out bytes.Buffer
    require.NoError(t, output.WriteJSON(&out, result))
    require.JSONEq(t, `{"schema_version":"1.0","tool_version":"1.0.0","command":"doctor","operation_id":"01JTEST","status":"passed","exit_code":0,"summary":"Environment is ready.","facts":[],"warnings":[],"blockers":[],"changes":[],"evidence":[],"next_actions":[],"approval":{"required":false,"class":"A","reason":""},"timing_ms":0}`, out.String())
}
```

- [ ] **Step 2: Run the focused test and verify the missing-package failure**

Run: `cd cli && go test ./internal/output -run TestWriteJSONUsesStableEnvelope -v`

Expected: FAIL because `domain.Result` and `output.WriteJSON` do not exist.

Initialize `cli/go.mod` with module `fullstack-orchestrator/cli`, `go 1.26.0`, and the exact Cobra, YAML, JSON Schema, and Testify versions listed in the Tech Stack header.

- [ ] **Step 3: Add the exact result types and encoder**

```go
type Status string

const (
    StatusPassed Status = "passed"
    StatusWarning Status = "warning"
    StatusBlocked Status = "blocked"
    StatusFailed Status = "failed"
    StatusUnknown Status = "unknown"
    StatusPartial Status = "partial"
    StatusApprovalRequired Status = "approval_required"
)

type Approval struct { Required bool `json:"required"`; Class string `json:"class"`; Reason string `json:"reason"` }
type Item struct { Code string `json:"code"`; Message string `json:"message"`; Refs []string `json:"refs,omitempty"` }
type Result struct {
    SchemaVersion string `json:"schema_version"`; ToolVersion string `json:"tool_version"`
    Command string `json:"command"`; OperationID string `json:"operation_id"`; Status Status `json:"status"`
    ExitCode int `json:"exit_code"`; Summary string `json:"summary"`
    Facts []Item `json:"facts"`; Warnings []Item `json:"warnings"`; Blockers []Item `json:"blockers"`
    Changes []Item `json:"changes"`; Evidence []Item `json:"evidence"`; NextActions []Item `json:"next_actions"`
    Approval Approval `json:"approval"`; TimingMS int64 `json:"timing_ms"`
}

func WriteJSON(w io.Writer, r domain.Result) error {
    if r.Facts == nil { r.Facts = []domain.Item{} }; if r.Warnings == nil { r.Warnings = []domain.Item{} }
    if r.Blockers == nil { r.Blockers = []domain.Item{} }; if r.Changes == nil { r.Changes = []domain.Item{} }
    if r.Evidence == nil { r.Evidence = []domain.Item{} }; if r.NextActions == nil { r.NextActions = []domain.Item{} }
    enc := json.NewEncoder(w); enc.SetEscapeHTML(false); return enc.Encode(r)
}
```

- [ ] **Step 4: Add `result.schema.json`, initialize Cobra, and return domain exit codes**

The root command must set `SilenceUsage: true`, render human output by default, select `WriteJSON` for `--json`, and call `os.Exit(result.ExitCode)`. Define codes `0,2,3,4,5,6,7,8`; reserve `1`.

- [ ] **Step 5: Run validation and commit**

Run: `cd cli && gofmt -w . && go test ./... && go vet ./...`

Expected: PASS with no vet findings.

Commit: `feat(cli): establish stable command result envelope`

## Task 2: Implement schema loading, canonical IDs, and cross-platform fingerprints

**Files:**
- Create: `schemas/manifest.schema.json`
- Create: `schemas/workspaces.schema.json`
- Create: `schemas/spec.schema.json`
- Create: `schemas/contract-registry.schema.json`
- Create: `cli/internal/schema/loader.go`
- Create: `cli/internal/schema/validate.go`
- Create: `cli/internal/context/fingerprint.go`
- Create: `cli/internal/context/fingerprint_test.go`

**Interfaces:**
- Consumes: `domain.Result` from Task 1.
- Produces: `schema.LoadYAML[T any](path string) (T, error)`, `schema.Validate(kind string, value any) []domain.Item`, `context.Fingerprint(kind string, data []byte) (string, error)`.

- [ ] **Step 1: Write failing normalization tests**

```go
func TestFingerprintNormalizesLineEndingsAndYAMLKeyOrder(t *testing.T) {
    a, err := context.Fingerprint("yaml", []byte("id: policy.account\r\nstatus: approved\r\n"))
    require.NoError(t, err)
    b, err := context.Fingerprint("yaml", []byte("status: approved\nid: policy.account\n"))
    require.NoError(t, err)
    require.Equal(t, a, b)
}

func TestFingerprintRejectsDuplicateYAMLKeys(t *testing.T) {
    _, err := context.Fingerprint("yaml", []byte("id: one\nid: two\n"))
    require.ErrorContains(t, err, "duplicate key")
}
```

- [ ] **Step 2: Run and observe failure**

Run: `cd cli && go test ./internal/context -run Fingerprint -v`

Expected: FAIL because `Fingerprint` is undefined.

- [ ] **Step 3: Implement strict YAML decoding and canonical SHA-256**

Decode into `yaml.Node`, reject duplicate mapping keys, recursively sort mapping keys, normalize scalar line endings to LF, encode canonical JSON, then return `sha256:` plus lowercase hex. Plain Markdown normalizes CRLF, removes trailing horizontal whitespace, and guarantees one final LF before hashing.

- [ ] **Step 4: Define exact schema invariants**

Schemas must require `schema_version`, stable lowercase dot ID pattern `^[a-z][a-z0-9]*(\.[a-z][a-z0-9-]*)+$`, enumerated status, integer revision `>=1`, unique references, workspace kind `root|directory|submodule|external`, and secrets prohibited via property names matching `token|password|secret|private_key`.

- [ ] **Step 5: Add valid/invalid golden fixtures and run all tests**

Run: `cd cli && go test ./internal/schema ./internal/context -v`

Expected: valid fixtures PASS; duplicate key, invalid ID, secret field, and unknown property fixtures produce stable error codes.

Commit: `feat(schema): validate canonical project metadata`

## Task 3: Build context discovery, index, impact graph, and stale propagation

**Files:**
- Create: `cli/internal/context/root.go`
- Create: `cli/internal/context/index.go`
- Create: `cli/internal/context/graph.go`
- Create: `cli/internal/context/refresh.go`
- Create: `cli/internal/context/refresh_test.go`
- Create: `testdata/projects/context-valid/**`
- Create: `testdata/projects/context-conflict/**`

**Interfaces:**
- Consumes: schema and fingerprint interfaces from Task 2.
- Produces: `context.FindRoot(start string) (string, error)`, `context.Refresh(ctx context.Context, root string, mode RefreshMode) (Snapshot, []domain.Item)`, `Snapshot.Index`, `Snapshot.Impact`, `Snapshot.Stale`.

- [ ] **Step 1: Write the failing source-precedence and stale tests**

```go
func TestRefreshMarksGeneratedSummaryStaleWhenSourceChanges(t *testing.T) {
    root := fixture.Copy(t, "context-valid")
    fixture.Replace(t, filepath.Join(root, "specs/policies/account.md"), "limit: 6", "limit: 7")
    got, issues := contextpkg.Refresh(context.Background(), root, contextpkg.ReadOnly)
    require.Empty(t, errorsOnly(issues))
    require.Contains(t, got.Stale, "docs.generated.current")
    require.Contains(t, got.Stale, "scenario.account.rate-limited")
}

func TestRefreshDoesNotLetTaskTitleOverrideApprovedPolicy(t *testing.T) {
    root := fixture.Copy(t, "context-conflict")
    got, _ := contextpkg.Refresh(context.Background(), root, contextpkg.ReadOnly)
    require.Equal(t, "specs/policies/account.md", got.Index["policy.account.rate-limit"].Path)
    require.Contains(t, got.Unknown, "work.GH-12.semantic-conflict")
}
```

- [ ] **Step 2: Run and verify failure**

Run: `cd cli && go test ./internal/context -run Refresh -v`

Expected: FAIL because root discovery and graph code do not exist.

- [ ] **Step 3: Implement root discovery and authored-document indexing**

Walk upward without crossing a filesystem volume boundary, select the nearest `.harness/manifest.yaml`, reject symlink/junction escape, scan only declared authored roots, parse metadata, reject duplicate stable IDs, and build sorted `ID -> path/revision/fingerprint/refs/status` entries.

- [ ] **Step 4: Implement deterministic graph and stale breadth-first propagation**

Create directed edges from `refs`, contract providers/consumers, workspace dependencies, UI coverage, and evidence source fingerprints. Seed stale nodes from fingerprint mismatch; breadth-first traverse impact edges; preserve explicit `unknown` when an external source cannot be checked. Sort every map-derived output before JSON encoding.

- [ ] **Step 5: Expose `context refresh|audit|pack` and verify read-only mode**

Run: `cd cli && go test ./... && go run ./cmd/orchestrator context audit --root ../testdata/projects/context-valid --json`

Expected: exit `0`, deterministic index counts, no project file changes under `git diff --exit-code`.

Commit: `feat(context): rebuild project state from canonical sources`

## Task 4: Add approval policy and crash-safe plan/apply operations

**Files:**
- Create: `cli/internal/policy/approval.go`
- Create: `cli/internal/policy/approval_test.go`
- Create: `cli/internal/operation/plan.go`
- Create: `cli/internal/operation/journal.go`
- Create: `cli/internal/operation/apply.go`
- Create: `cli/internal/operation/recovery_test.go`

**Interfaces:**
- Produces: `policy.Classify(Action, Consent) Decision`, `operation.Plan`, `operation.Apply(ctx, Plan) domain.Result`, and idempotency receipts.

- [ ] **Step 1: Write table-driven approval tests**

```go
func TestApprovalClasses(t *testing.T) {
    cases := []struct{ action policy.Action; want string; always bool }{
        {policy.ReadStatus, "A", false}, {policy.WriteRequestedCode, "B", false},
        {policy.AddSubmodule, "C", false}, {policy.PushBranch, "C", false},
        {policy.ForcePush, "D", true}, {policy.PublishProduction, "D", true},
        {policy.SendSecretExternal, "D", true},
    }
    for _, tc := range cases { got := policy.Classify(tc.action, policy.Consent{}); require.Equal(t, tc.want, got.Class); require.Equal(t, tc.always, got.AlwaysConfirm) }
}
```

- [ ] **Step 2: Write a failing fault-injection recovery test**

The test creates a three-file plan, injects failure after the first atomic rename, re-runs with the same operation ID, and asserts one final copy of every file, no partial temp files, and one completed receipt.

- [ ] **Step 3: Implement scoped consent and non-bypassable D actions**

Consent matching must require objective, repository, action, and expiry match. `--yes` can satisfy an already-authorized B plan only; C requires explicit current-scope authorization and D always returns exit `3` until an exact target approval receipt is supplied.

- [ ] **Step 4: Implement plan fingerprint, atomic writes, and journal recovery**

Use same-directory temporary files with `0600`, `fsync`, atomic rename, directory sync where supported, and an append-only JSONL journal under `.harness/local/operations/<id>/`. Revalidate the initial state fingerprint before the first mutation and each remote target before an external write.

- [ ] **Step 5: Run fault, race, and unit tests**

Run: `cd cli && go test -race ./internal/policy ./internal/operation -v`

Expected: all cases PASS; repeated apply produces the existing receipt without duplicate mutation.

Commit: `feat(operation): add scoped approval and recoverable apply`

## Task 5: Implement safe Git, submodule, worktree, and actual-state diagnostics

**Files:**
- Create: `cli/internal/gitx/runner.go`
- Create: `cli/internal/gitx/status.go`
- Create: `cli/internal/gitx/submodule.go`
- Create: `cli/internal/gitx/worktree.go`
- Create: `cli/internal/gitx/git_integration_test.go`
- Create: `testdata/git/fixture.go`

**Interfaces:**
- Consumes: operation and approval engine from Task 4.
- Produces: `gitx.Inspect(ctx, root) State`, `gitx.PlanWorkspaceSync(State) operation.Plan`, `gitx.PlanWorktree(Change) operation.Plan`.

- [ ] **Step 1: Write real-repository integration cases**

Create temporary bare remotes and clones for clean, dirty, ahead, behind, diverged, detached submodule, missing submodule, pointer mismatch, and parallel worktree. Assert `Inspect` never changes reflog, index, worktree, or submodule HEAD.

- [ ] **Step 2: Run and verify missing implementation**

Run: `cd cli && go test ./internal/gitx -run TestInspect -v`

Expected: FAIL because `Inspect` is undefined.

- [ ] **Step 3: Implement a shell-free Git runner**

Use `exec.CommandContext(ctx, "git", args...)` with explicit working directory, a minimal inherited environment, bounded stdout/stderr, timeout, and redaction. Disallow `-c core.sshCommand`, unsafe protocols, hooksPath overrides, and commands outside an adapter allowlist.

- [ ] **Step 4: Implement exact-pointer sync planning**

Read `.gitmodules`, root gitlink SHA, submodule HEAD/branch/dirty state, and remote tracking. A clean missing/old submodule may plan `git submodule update --init --recursive -- <path>` at the pinned SHA. Detached local changes, divergence, unsafe URL, nested submodule, or dirty root pointer return blocker `4`; never use `update --remote`.

- [ ] **Step 5: Implement worktree planning and branch conventions**

Validate `<type>/<description>` or `<type>/<work-id>-<description>`, reject AI markers only when a project policy explicitly lists them, create worktrees outside the repository root, and record their path/branch/claim without deleting them automatically.

- [ ] **Step 6: Run Git topology suite on macOS and Windows CI**

Run: `cd cli && go test -race ./internal/gitx -v`

Expected: all topologies return stable state and blocker codes; no hidden stash/reset/rebase/pull occurs.

Commit: `feat(git): inspect and plan safe workspace coordination`

## Task 6: Implement work items, claims, semantic conflict detection, and change bundles

**Files:**
- Create: `schemas/work-item.schema.json`
- Create: `schemas/claim.schema.json`
- Create: `schemas/change.schema.json`
- Create: `cli/internal/domain/work.go`
- Create: `cli/internal/policy/conflict.go`
- Create: `cli/internal/policy/conflict_test.go`
- Create: `cli/internal/project/work.go`

**Interfaces:**
- Produces: `policy.CheckConflict(Candidate, []Claim, context.Snapshot) ConflictReport`, `project.StartWork(Request) operation.Plan`, and `project.FinishWork(Request) domain.Result`.

- [ ] **Step 1: Write conflict matrix tests**

Cover path overlap, same policy ID/different files, same contract, same DB entity/migration slot, UI route/flow overlap, dependency major change, root pointer order, expired claim, provider unavailable, and clear independent work. Expected levels are exactly `clear`, `coordinate`, `block`, `unknown`.

- [ ] **Step 2: Run and verify failure**

Run: `cd cli && go test ./internal/policy -run Conflict -v`

Expected: FAIL because conflict types do not exist.

- [ ] **Step 3: Implement normalized scopes and lease rules**

Normalize repository, workspace, slash-separated path glob, stable IDs, contract IDs, DB entity/migration parent, UI route/flow, dependency manifest, baseline, branch, owner, start, and expiry. Claims show intent; they do not provide distributed locking. For the Git fallback, read active claim records from fetched remote refs without checkout, require the feature branch claim commit to be pushed for collaborative visibility, and keep unobservable remote state `unknown`.

- [ ] **Step 4: Implement compatibility-aware change bundles**

Represent workspace PRs/commits, root integration branch, contract versions, provider/consumer order, pointer update, verification, rollback, and release target. Breaking changes require side-by-side versions unless an approved maintenance-window plan exists.

- [ ] **Step 5: Expose work/conflict commands and run tests**

Run: `cd cli && go test ./internal/policy ./internal/project -v`

Expected: every conflict scenario returns the designed level and a concrete safe next action.

Commit: `feat(work): coordinate claims and cross-repository changes`

## Task 7: Add contract, DBML, dbdiagram, and UI-source workflows

**Files:**
- Create: `cli/internal/contract/check.go`
- Create: `cli/internal/contract/compatibility.go`
- Create: `cli/internal/database/dbml.go`
- Create: `cli/internal/database/dbdiagram.go`
- Create: `cli/internal/ui/import.go`
- Create: `cli/internal/ui/coverage.go`
- Create: `cli/internal/contract/compatibility_test.go`
- Create: `cli/internal/database/dbdiagram_test.go`
- Create: `cli/internal/ui/import_test.go`

**Interfaces:**
- Produces: `contract.Compare(old, next) Report`, `database.PullPlan(config) operation.Plan`, `database.SemanticDiff(a, b) Diff`, `ui.ImportPlan(source) operation.Plan`, `ui.CheckCoverage(snapshot) Report`.

- [ ] **Step 1: Write failing compatibility tests**

Test additive optional field as compatible, required field/removal/type narrowing/error semantic change as breaking, retry/idempotency policy change as semantic breaking, and version side-by-side as coordinated.

- [ ] **Step 2: Write dbdiagram isolation and secret tests**

Fake the adapter executable and assert `pull` targets `.harness/local/dbdiagram/<operation-id>/`, canonical DBML is byte-identical before approval, the token never appears in argv/output/receipt, and a direct remote semantic change returns approval-required with linked policy questions.

- [ ] **Step 3: Write malicious UI import tests**

Use archives containing `../escape`, symlink escape, oversized decompression, executable script, missing license, and embedded token. Assert rejection or quarantined warning before canonical UI files change.

- [ ] **Step 4: Implement deterministic parsers and adapter plans**

Parse contracts through registered standards and normalized obligations; parse DBML into table/column/relation/index/note entities; invoke official dbdiagram `init/push/pull` only through the approved adapter; record UI source ID, kind, authority, version, license, hash, and journey coverage.

- [ ] **Step 5: Expose contract/db/ui commands and run tests**

Run: `cd cli && go test -race ./internal/contract ./internal/database ./internal/ui -v`

Expected: compatibility and import safety cases PASS; no canonical write occurs during pull/import plan.

Commit: `feat(boundaries): verify contracts database and UI sources`

## Task 8: Generate and adopt framework-neutral projects

**Files:**
- Create: `cli/internal/project/draft.go`
- Create: `cli/internal/project/init.go`
- Create: `cli/internal/project/adopt.go`
- Create: `cli/internal/project/generate.go`
- Create: `cli/internal/project/project_e2e_test.go`
- Create: `templates/project/**`

**Interfaces:**
- Produces: `project.CreateDraft`, `project.PlanInit`, `project.PlanAdopt`, `project.Render`, with exact structure from design document 02.

- [ ] **Step 1: Write new-project E2E failure case**

The test starts in an empty parent, writes a discovery draft, approves product summary and repository name, applies init, then asserts `AGENTS.md`, `.agents/skills/use-project-harness/SKILL.md`, `.harness`, `specs`, `contracts`, and `docs` exist; framework-specific source directories do not exist.

- [ ] **Step 2: Write existing-project non-destructive E2E case**

Seed custom `README.md`, `AGENTS.md`, `.gitignore`, code, Git history, and a dirty user file. Plan adoption; assert no file changes. Apply with managed-section approval; assert custom content, history, topology, and dirty file remain unchanged.

- [ ] **Step 3: Implement draft persistence and normalized discovery**

Write `.harness-drafts/<ulid>/manifest.yaml`, `state.yaml`, normalized product summary, decisions, and open questions using atomic operations. Never store raw conversation or secret values. Migrate the draft only after generated source fingerprints validate.

- [ ] **Step 4: Implement template render and managed-section merge**

Embed templates in the Go binary. Render UTF-8/LF logical files; merge only explicit markers in existing `README.md`/`AGENTS.md`; produce a blocking diff when existing `.editorconfig`, `.gitattributes`, or `.gitignore` conflicts semantically.

- [ ] **Step 5: Add exact generated state files from design 03**

Generate `.harness/sources.yaml`, authored YAML policies, empty active work directories where Git requires them via `.gitkeep` only when necessary, tracked checkpoint `context-index.json` and `impact-graph.json`, and Git-ignored `.harness/local/state/current.json` after validation.

- [ ] **Step 6: Run golden and E2E tests**

Run: `cd cli && go test ./internal/project -update=false -v && git diff --exit-code -- testdata`

Expected: new and adopt fixtures PASS; no golden drift.

Commit: `feat(project): initialize and adopt neutral project harnesses`

## Task 9: Package focused Skills, repo-local fallback, and trusted Hooks

**Files:**
- Create: `.codex-plugin/plugin.json`
- Create: `.agents/plugins/marketplace.json`
- Create: `compatibility.json`
- Create: `skills/{start-project,resume-project,find-next-work,plan-project-change,start-project-work,manage-contract-change,design-project-database,import-project-ui,integrate-project-work,prepare-project-release,handoff-project-work,audit-project-context}/SKILL.md`
- Create: `references/{lifecycle,approval,context-recovery}.md`
- Create: `hooks/hooks.json`
- Create: `scripts/validate-plugin.sh`
- Test: `testdata/plugin/**`

**Interfaces:**
- Consumes: stable CLI commands and JSON from Tasks 1–8.
- Produces: installable Codex Plugin and generated repo-local fallback Skill.

- [ ] **Step 1: Write manifest and Skill validation tests**

The validator must assert only `.codex-plugin/plugin.json` is inside `.codex-plugin`, all Skill directories have one `SKILL.md`, frontmatter names equal directory names, descriptions state trigger and action, every relative reference exists, and no Skill duplicates policy text from the canonical references.

- [ ] **Step 2: Write behavior fixtures for each Skill**

Each fixture supplies a user utterance and CLI JSON response, then asserts selected Skill, required references, command, whether a question is allowed, and the final evidence wording. Include “너 내용 잊은 것 같아” → `audit-project-context` → `context audit`.

- [ ] **Step 3: Author the twelve focused Skills**

Every `SKILL.md` must: refresh context before mutation; ask one material question at a time; put the recommended option first with free-form other input; avoid asking discoverable facts; route deterministic work to the CLI; distinguish fact/stale/unknown; and never claim completion without evidence.

- [ ] **Step 4: Add optional Hook behavior**

SessionStart emits only project root and stale-context notice; PostCompact emits only the requirement to run context audit before mutation. Both must refuse untrusted projects and perform no write, install, Git mutation, external call, or secret access.

- [ ] **Step 5: Validate using the current Codex plugin validator**

Run: `bash scripts/validate-plugin.sh && cd cli && go test ./...`

Expected: manifest, marketplace, Skills, references, Hook schema, missing-CLI fallback, and trusted/untrusted fixtures PASS. Add a manifest `hooks` field only if the target validator accepts it.

Commit: `feat(plugin): package project orchestration skills`

## Task 10: Add provider capability negotiation and first-party adapters

**Files:**
- Create: `cli/internal/provider/provider.go`
- Create: `cli/internal/provider/registry.go`
- Create: `cli/internal/provider/github/*.go`
- Create: `cli/internal/provider/gitgeneric/*.go`
- Create: `cli/internal/provider/dbdiagram/*.go`
- Create: `cli/internal/provider/provider_contract_test.go`

**Interfaces:**
- Produces: `provider.Adapter` with `Descriptor`, `Discover`, `Capabilities`, `Health`, `Plan`, `Execute`, `Normalize`, and `Receipt` methods.

- [ ] **Step 1: Write adapter contract tests**

Run every adapter through no-auth, offline, read-only, approval-required write, duplicate operation, rate limit, malformed response, secret redaction, and unsupported capability cases. Unsupported hierarchy/dependency must return a capability warning, not fabricated remote state.

- [ ] **Step 2: Implement the interface and registry**

Use typed capability constants for read, write, hierarchy, dependency, draft review, release, and diagram sync. Select exactly one live task-status source. Registry priority is explicit project config, then detected provider, then local Git fallback.

- [ ] **Step 3: Implement GitHub and generic Git adapters**

GitHub supports Issue/sub-issue/dependency/PR/release capabilities through a replaceable client interface; generic Git supports fetch/status/branch/push/tag only. External writes require operation idempotency and approval receipts.

- [ ] **Step 4: Integrate dbdiagram and local fallback adapters**

Expose official CLI capability based on detected version, token availability, and project ID. Local task fallback uses validated `.harness/work/items` and claims; status must not be simultaneously writable in another provider.

- [ ] **Step 5: Run provider contracts**

Run: `cd cli && go test -race ./internal/provider/... -v`

Expected: all adapters satisfy identical safety and degraded-mode assertions.

Commit: `feat(provider): negotiate external collaboration capabilities`

## Task 11: Implement production gates, immutable RC, and release verification

**Files:**
- Create: `schemas/release-candidate.schema.json`
- Create: `cli/internal/release/gate.go`
- Create: `cli/internal/release/candidate.go`
- Create: `cli/internal/release/publish.go`
- Create: `cli/internal/release/release_test.go`
- Create: `testdata/releases/**`

**Interfaces:**
- Produces: `release.Verify`, `release.CreateCandidate`, `release.VerifyCandidate`, `release.PlanPublish`.

- [ ] **Step 1: Write failing RC immutability tests**

Create an RC from root/workspace commits, plugin/CLI digests, schema/adapter versions, SBOM/provenance/signatures, gate receipts, and docs fingerprint. Mutate each input one at a time and assert `VerifyCandidate` fails with the exact changed field.

- [ ] **Step 2: Write release blocker tests**

Cases: flaky required check, manual-only critical verification, unsigned artifact, unverified migration rollback, unsafe Hook, missing Windows/macOS journey, Plugin-less continuation failure, user-validation SHA mismatch, and unowned warning. Every case must block publish.

- [ ] **Step 3: Implement gate aggregation and candidate digest**

Sort every source by stable ID, require blocker count zero, require warning owner/rationale, compute one SHA-256 manifest digest, and write it atomically. `rc create` is C; `release publish` remains D even with standing consent.

- [ ] **Step 4: Implement publish planning without host side effects**

Plan protected tag, clean reproducible build, test receipt check, SBOM/provenance/signature, GitHub Release, Plugin marketplace update, Homebrew/WinGet update, installation smoke tests, release notes, rollback, and support links. Adapter execution is allowed only after exact RC approval.

- [ ] **Step 5: Run release suite**

Run: `cd cli && go test -race ./internal/release -v`

Expected: immutable RC passes unchanged, every mutated input and blocker prevents publish with exit `5` or approval-required `3`.

Commit: `feat(release): verify immutable production candidates`

## Task 12: Add localization, documentation, examples, and privacy-safe diagnostics

**Files:**
- Create: `locales/en/messages.json`
- Create: `locales/ko/messages.json`
- Create: `cli/internal/output/localize.go`
- Create: `cli/internal/output/localize_test.go`
- Create: `docs/getting-started/{en,ko}.md`
- Create: `docs/concepts/{en,ko}.md`
- Create: `docs/guides/{new-project,existing-project,submodules,dbdiagram,release}-{en,ko}.md`
- Create: `docs/security/{threat-model,privacy}-{en,ko}.md`
- Create: `examples/starter/**`
- Create: `examples/multi-repo/**`
- Create: `cli/internal/diagnostic/export.go`
- Create: `cli/internal/diagnostic/export_test.go`

**Interfaces:**
- Produces: locale-complete human output and `doctor --export` diagnostic archive.

- [ ] **Step 1: Write locale parity test**

Parse both message catalogs; require identical sorted keys, placeholder names, severity, and documentation section IDs. Missing Korean text or English-only new key fails CI.

- [ ] **Step 2: Write diagnostic privacy tests**

Seed tokens, home paths, repository source, prompts, Git remote credentials, and provider output. Export and assert none are present; include versions, architecture, stable errors, redacted state, and operation receipts.

- [ ] **Step 3: Implement localization and safe export**

Select explicit `--locale`, then project locale, then OS locale, then English. Keep JSON stable English machine codes while translating human summaries. Replace home/root paths with symbolic labels and omit file content by default.

- [ ] **Step 4: Write both end-to-end guides and executable examples**

Document exact natural-language entry, CLI equivalent, generated files, approvals, failure recovery, Plugin-less fallback, and release flow. Every command in docs runs against the examples in CI.

- [ ] **Step 5: Run parity, docs, and example tests**

Run: `cd cli && go test ./internal/output ./internal/diagnostic ./... && go run ./cmd/orchestrator doctor --root ../examples/starter --json`

Expected: catalogs match, diagnostic is privacy-safe, examples pass context audit.

Commit: `docs(product): add verified English and Korean journeys`

## Task 13: Build signed cross-platform packages and the 1.0.0 release pipeline

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/security.yml`
- Create: `.github/workflows/release.yml`
- Create: `.goreleaser.yaml`
- Create: `packaging/homebrew/orchestrator.rb`
- Create: `packaging/winget/*.yaml`
- Create: `SECURITY.md`
- Create: `SUPPORT.md`
- Create: `CONTRIBUTING.md`
- Create: `GOVERNANCE.md`
- Create: `LICENSE`

**Interfaces:**
- Consumes: all earlier code, plugin, docs, and RC manifest.
- Produces: signed macOS/Windows artifacts, SBOM, provenance, marketplace package, and reproducible 1.0.0 publication.

- [ ] **Step 1: Create CI matrix with required evidence**

Matrix: `macos-14/arm64`, `macos-13/x86_64`, `windows-11/x86_64`, `windows-11/arm64` using available native or verified emulated runners. Run unit, race where supported, fuzz smoke, real-Git integration, plugin validation, schema golden, docs command, install/update/uninstall, and Plugin-less E2E.

- [ ] **Step 2: Add supply-chain checks**

Use pinned actions by commit SHA, least-privilege permissions, OIDC keyless signing, dependency/license/secret/static scans, Syft SBOM, SLSA-compatible provenance, and checksum verification. Fork PR jobs receive no release or provider secret.

- [ ] **Step 3: Configure deterministic artifacts**

GoReleaser builds `darwin_amd64`, `darwin_arm64`, `windows_amd64`, and `windows_arm64` with `CGO_ENABLED=0`, trimmed paths, fixed build metadata, zip archives, checksums, SBOM, and signature. Windows MSI is signed using the protected release environment.

- [ ] **Step 4: Add guarded release workflow**

Accept only an approved RC digest, verify source tag and user-validation receipt match, rebuild in a clean environment, compare reproducible digests, publish immutable GitHub artifacts, update marketplace/Homebrew/WinGet, run clean install smoke tests, and stop before publish without D approval.

- [ ] **Step 5: Rehearse rollback and incident response**

In a staging repository, publish a signed candidate, install, upgrade a schema fixture, roll back binary with compatible schema, restore an incompatible migration backup, revoke a test signing identity, and process a private security report. Save receipts without secrets.

- [ ] **Step 6: Perform public-name identity freeze**

With the user-approved name, verify repository/package/domain/trademark confusion, replace only the working binary/module/plugin/marketplace/package labels, rerun every test and install path, and record the stable public IDs before creating any public package.

- [ ] **Step 7: Create and verify the 1.0.0 RC**

Run: `orchestrator verify release --json`, then `orchestrator rc create --version 1.0.0 --json`, perform user journeys on the exact artifacts, then `orchestrator rc verify --json`.

Expected: blocker count zero, warning count zero or every warning explicitly owned and approved, technical/user receipts reference the same RC digest.

- [ ] **Step 8: Publish only after exact final approval**

Run: `orchestrator release publish --rc <approved-rc-digest> --operation <approved-operation-id>`.

Expected: immutable GitHub 1.0.0 release, installable Codex marketplace plugin, working Homebrew and WinGet installs, verified checksums/signatures/SBOM/provenance, and support/rollback links.

Commit: `ci(release): publish verified cross-platform artifacts`

---

## Self-review result

- Spec coverage: lifecycle, project structure, source precedence, context recovery, Git/submodules/worktrees, conflict claims, approvals, providers, DBML/dbdiagram, external UI, Skills/Plugin/Hook, macOS/Windows CLI, TDD, security, RC, user verification, packaging, and operations each map to at least one task.
- Placeholder scan: the public name is not represented as an unresolved implementation token; `orchestrator` and `fullstack-orchestrator` are exact working identifiers with an explicit identity-freeze task.
- Type consistency: all later tasks consume the `domain.Result`, context snapshot, operation plan, adapter, and release interfaces introduced earlier; command and exit-code names match design document 07.
- Execution order: Tasks 1–4 form the safety core; Tasks 5–8 provide a usable project vertical slice; Tasks 9–10 add distribution and external integrations; Tasks 11–13 complete production release.
