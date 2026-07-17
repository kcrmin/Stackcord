# Focused Product Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refocus the existing implementation into a production-quality AI-guided full-stack project harness with repeatable discovery, durable clone continuation, Git/submodule collaboration, semantic conflict prevention, five Skills, and core-versus-strict release profiles.

**Architecture:** Preserve the verified Go safety, context, Git, contract, database, UI, and operation modules. Replace broad platform surfaces with one structured discovery checkpoint, a minimal generated harness, five intent Skills, truthful external-tool boundaries, and one profile-aware release command. Keep strict supply-chain publication only as an optional product profile.

**Tech Stack:** Go 1.26.5, Cobra 1.10.2, strict YAML/JSON, JSON Schema Draft 2020-12, Agent Skills, Codex Plugin, GitHub Actions, GoReleaser for this product's optional strict publication.

## Global Constraints

- Work only in the isolated repository under Soomgil until final verification.
- Preserve commits `ff02bfb` and `b6540e1` as recovery points.
- The AI owns flexible conversation and judgment; the CLI owns deterministic state and safety checks.
- Generated projects remain framework, language, database, cloud, and task-provider neutral.
- Users never need to manage operation IDs, claims, receipts, or CLI commands directly.
- Raw conversation, tone, prompts, and credentials are never persisted.
- Behavior, bug, contract, migration, and UI interaction changes use TDD.
- No hidden fetch, pull, rebase, stash, reset, clean, force-push, external write, or publication.
- macOS arm64/amd64 and Windows arm64/amd64 remain supported.
- Public naming and irreversible publication remain external blockers.

---

## Task 1: Replace one-shot drafts with repeatable normalized discovery checkpoints

**Files:**
- Create: `schemas/discovery.schema.json`
- Create: `cli/internal/schema/definitions/discovery.schema.json`
- Create: `cli/internal/project/checkpoint.go`
- Create: `cli/internal/project/checkpoint_test.go`
- Modify: `cli/internal/project/draft.go`
- Modify: `cli/internal/command/project.go`
- Modify: `cli/internal/command/root_test.go`

**Interfaces:**
- Produces `project.DiscoveryCheckpoint`, `project.PlanCheckpoint(request CheckpointRequest) (operation.Plan, error)`, and `orchestrator project checkpoint --parent --id --input [--apply]`.
- The revision is read from the existing draft and incremented by the CLI; the user does not supply it.

- [ ] **Step 1: Write a failing repeat-checkpoint test**

```go
func TestCheckpointRevisesOneDiscoveryWithoutRawConversation(t *testing.T) {
    parent := t.TempDir()
    first := validCheckpoint("Account recovery", "Which recovery proof is allowed?")
    plan, err := project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JDISCOVERY", Locale: "ko", Checkpoint: first})
    require.NoError(t, err)
    require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

    second := validCheckpoint("Account recovery for members", "How long are recovery attempts retained?")
    plan, err = project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JDISCOVERY", Locale: "ko", Checkpoint: second})
    require.NoError(t, err)
    require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
    require.Contains(t, readState(t, parent), "revision: 2")
    require.NotContains(t, readDraftTree(t, parent), "User said")
}
```

- [ ] **Step 2: Run `cd cli && go test ./internal/project -run Checkpoint -v` and verify failure because `PlanCheckpoint` does not exist.**

- [ ] **Step 3: Implement strict checkpoint input and revisioned atomic plans** with fields `summary`, `current_focus`, `roles`, `journeys`, `capabilities`, `policies`, `scenarios`, `quality`, `ui_coverage`, `technology_needs`, `decisions`, `assumptions`, and `open_questions`. Write normalized YAML/Markdown under `.harness-drafts/<id>/`; derive operation ID `checkpoint-<id>-r<revision>`.

- [ ] **Step 4: Add the command and remove `project draft` from the command surface.** The command accepts a strict YAML or JSON input file, reports the new revision, and plans unless `--apply` is present.

- [ ] **Step 5: Run `cd cli && go test ./internal/project ./internal/command -v` and commit.**

Commit: `feat(discovery): add repeatable normalized checkpoints`

## Task 2: Generate only the minimal durable project harness

**Files:**
- Modify: `cli/internal/project/generate.go`
- Modify: `cli/internal/project/adopt.go`
- Modify: `cli/internal/project/project_e2e_test.go`
- Modify: `templates/project/**`
- Modify: `schemas/manifest.schema.json`
- Create: `schemas/profile.schema.json`
- Create: `cli/internal/schema/definitions/profile.schema.json`

**Interfaces:**
- `project.PlanInit` and `project.PlanAdopt` retain their signatures.
- Generated `.harness/profile.yaml` declares `tdd: default`, `git.collaboration: strongly_recommended`, `git.release: required`, `task_source: git-local`, and `release: core`.

- [ ] **Step 1: Change the new-project E2E test to assert the exact generated file set.** Assert no `.harness/policies`, templates, integrations, lifecycle, baselines, gates, empty claims, release candidate, or placeholder directories exist.

- [ ] **Step 2: Run `cd cli && go test ./internal/project -run NewProject -v` and observe extra-file failure.**

- [ ] **Step 3: Remove generated policy copies and empty directories; add `profile.yaml`.** Keep only README/AGENTS/tooling, repo-local Skill/fallback, manifest/entry/profile/sources/workspaces/context state/provider, specs index, contract registry, and docs index.

- [ ] **Step 4: Ensure adoption remains non-destructive.** Existing README/AGENTS managed sections and ordered `.gitignore` rules remain preserved; existing authored harness files win unless schema-invalid.

- [ ] **Step 5: Verify template parity with generated content and run `cd cli && go test ./internal/project ./internal/context -v`.**

Commit: `refactor(project): generate a minimal durable harness`

## Task 3: Tighten authored context while retaining real Git collaboration

**Files:**
- Modify: `cli/internal/context/refresh.go`
- Modify: `cli/internal/context/refresh_test.go`
- Modify: `cli/internal/command/root.go`
- Modify: `cli/internal/command/root_test.go`
- Modify: `cli/internal/command/work.go`
- Delete: `cli/internal/provider/**`

**Interfaces:**
- Keep `context.Refresh`, `gitx.Inspect`, `gitx.ReadRemoteFiles`, semantic claims, and Git-local task files.
- Remove `context pack` and the unused provider registry/adapters.

- [ ] **Step 1: Add failing tests that reject invalid authored stable IDs, statuses, duplicate refs, and malformed contract metadata during context audit.**

- [ ] **Step 2: Add a failing command-surface test requiring `context audit|refresh` and rejecting `context pack`.**

- [ ] **Step 3: Validate every indexed authored document through the appropriate schema before accepting it into the context index.** Keep navigation indexes excluded and unknown external state explicit.

- [ ] **Step 4: Delete the unused provider packages and tests.** Keep task-source selection in the project profile/provider file: `git-local` is executable; any other configured source returns unavailable until a concrete connector is installed.

- [ ] **Step 5: Run `cd cli && go test -race ./internal/context ./internal/gitx ./internal/policy ./internal/command -v`.**

Commit: `refactor(core): keep deterministic project coordination only`

## Task 4: Consolidate the Plugin into five intent Skills

**Files:**
- Replace: `skills/*/SKILL.md`
- Replace: `testdata/plugin/behavior.json`
- Modify: `scripts/validate_plugin.py`
- Modify: `scripts/validate_plugin_test.py`
- Replace: `references/lifecycle.md`
- Replace: `references/approval.md`
- Modify: `references/context-recovery.md`
- Modify: `.codex-plugin/plugin.json`
- Modify: `templates/project/.agents/skills/use-project-harness/**`
- Modify: `cli/internal/project/generate.go`

**Interfaces:**
- Exposes exactly `start-project`, `continue-project`, `plan-project-work`, `coordinate-project-work`, and `recover-and-release-project`.
- Shared references become `workflow.md`, `safety.md`, and `context-recovery.md`.

- [ ] **Step 1: Replace behavior fixtures first.** Cover new/adopt, resume/next, feature/bug/TDD/worktree, contract/DBML/dbdiagram/UI/conflict/handoff/integration, forgotten context, core release, and strict release. Run `python3 scripts/validate_plugin_test.py` and observe missing Skill failures.

- [ ] **Step 2: Write one Skill at a time.** Each description contains triggers only; each body starts with context audit, hides internal nouns in normal user output, asks one material question at a time, and routes deterministic checks to the CLI.

- [ ] **Step 3: Replace duplicated references.** `workflow.md` contains discovery/TDD/integration heuristics, `safety.md` contains no-hidden-mutation rules, and `context-recovery.md` contains source precedence and fallback. Do not repeat A-D tables or lifecycle prose in Skills.

- [ ] **Step 4: Update the repo-local Skill to a compact universal entry.** It must work without the Plugin or CLI and tell the AI how to state reduced verification.

- [ ] **Step 5: Run the repository validator, official Skill validators, and official Plugin validator.**

Commit: `refactor(plugin): consolidate project workflows into five skills`

## Task 5: Replace enterprise-only default release gates with core and strict profiles

**Files:**
- Replace: `cli/internal/release/gate.go`
- Replace: `cli/internal/release/candidate.go`
- Replace: `cli/internal/release/release_test.go`
- Modify: `cli/internal/command/release.go`
- Modify: `cli/internal/command/root.go`
- Modify: `cli/internal/command/root_test.go`
- Replace: `schemas/release-candidate.schema.json`
- Modify: `testdata/releases/valid-input.json`
- Delete: `cli/internal/release/publish.go`

**Interfaces:**
- `release.Profile` is `core` or `strict-release`.
- `release.Input` always requires version, root/workspace commits, artifact digests, product/docs/contract fingerprints, TDD evidence, integration evidence, and conditional migration evidence.
- `release.StrictEvidence` is optional for core and required for strict.
- `release.UserValidation` is created after preparation and binds explicit confirmation to the candidate digest; `release verify` combines it with the current technical identities.
- CLI exposes only `release prepare|verify`.

- [ ] **Step 1: Write failing tests showing a valid core candidate passes without SBOM/signature/provenance and a strict candidate fails without them.**

- [ ] **Step 2: Write failing command-surface tests rejecting top-level `verify`, top-level `rc`, and `release publish`.**

- [ ] **Step 3: Implement profile-aware candidate validation and deterministic digest.** Verify the candidate manifest itself before comparing current inputs; changing any core or enabled strict identity blocks verification.

- [ ] **Step 4: Consolidate commands.** `release prepare` plans/applies the candidate and `release verify` compares it with current input. Internal operation IDs are derived from version and candidate digest.

- [ ] **Step 5: Run `cd cli && go test -race ./internal/release ./internal/command -v`.**

Commit: `refactor(release): separate core and strict verification`

## Task 6: Isolate this product's strict publication assets

**Files:**
- Create: `profiles/strict-release/README.md`
- Move: `packaging/**` to `profiles/strict-release/packaging/**`
- Move: `scripts/release/**` to `profiles/strict-release/scripts/**`
- Modify: `.github/workflows/release.yml`
- Modify: `.goreleaser.yaml`
- Modify: `scripts/validate_release_config.py`
- Modify: `README.md`
- Modify: `README.ko.md`

**Interfaces:**
- Default generated projects contain no publication assets.
- This product can still use the strict profile for its own public artifacts.

- [ ] **Step 1: Update release-config tests to require strict assets only under `profiles/strict-release`.** Run them and observe missing-path failures.

- [ ] **Step 2: Move files with history-preserving filesystem changes and update workflow paths.** Keep signing, SBOM, package-generation, and exact approval guards fail-closed.

- [ ] **Step 3: Update GoReleaser and validators to the new paths.** Ensure no default README text implies every project must publish all channels.

- [ ] **Step 4: Run all strict release script tests, `actionlint`, and `goreleaser check`.**

Commit: `refactor(release): isolate strict publication tooling`

## Task 7: Complete focused E2E scenarios and bilingual user documentation

**Files:**
- Create: `cli/internal/command/focused_e2e_test.go`
- Modify: `examples/starter/**`
- Modify: `examples/multi-repo/**`
- Replace: `README.md`
- Replace: `README.ko.md`
- Modify: `docs/getting-started/{en,ko}.md`
- Modify: `docs/concepts/{en,ko}.md`
- Modify: `docs/guides/{new-project,existing-project,submodules,dbdiagram,release}-{en,ko}.md`
- Create: `docs/guides/troubleshooting-en.md`
- Create: `docs/guides/troubleshooting-ko.md`
- Modify: `scripts/validate_docs.py`

**Interfaces:**
- Documentation leads with natural-language usage, not commands or internal file names.
- E2E fixtures prove checkpoint, init/adopt, clone recovery, compaction recovery routing, multi-repo/submodule, contract, DBML, UI, conflict, integration, and RC flows.

- [ ] **Step 1: Add executable focused E2E tests before changing docs.** Use real temporary Git repositories and archives; assert Plugin-less fallback contains enough information to resume.

- [ ] **Step 2: Run the E2E tests and fix any uncovered core behavior gaps with TDD.**

- [ ] **Step 3: Rewrite README and core guides around five natural-language journeys.** Include exact generated files, collaboration behavior, conflict outcomes, context recovery, core/strict differences, installation, testing, and external blockers.

- [ ] **Step 4: Add troubleshooting for forgotten context, dirty/diverged Git, submodule mismatch, unobservable task source, invalid imported UI, stale DBML, and RC mismatch.**

- [ ] **Step 5: Run documentation parity, example audits, and all commands embedded in core guides.**

Commit: `docs(product): document the focused user experience`

## Task 8: Final production verification and export readiness

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/security.yml`
- Modify: `scripts/validate_release_config.py`
- Create: `docs/release-readiness.md`

**Interfaces:**
- Produces a clean independent Git repository ready to copy out of Soomgil after all local checks.

- [ ] **Step 1: Ensure CI runs unit/integration tests, race where supported, fuzz smoke, real Git tests, Plugin validation, Plugin-less E2E, docs parity, and four cross-build targets.**

- [ ] **Step 2: Run fresh local verification:**

```sh
cd cli
go test ./...
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go test ./internal/context -run '^$' -fuzz FuzzFingerprint -fuzztime 15s
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

- [ ] **Step 3: Run repository verification:** Plugin validators, security scan, docs parity, strict release script tests, actionlint, GoReleaser check, snapshot release, and macOS/Windows amd64/arm64 cross-builds.

- [ ] **Step 4: Smoke-test a native binary against new project, existing adoption, context audit, Git inspect, and core RC.** Verify no files outside temporary fixtures or the isolated product repository changed.

- [ ] **Step 5: Review `git diff main...HEAD`, run `git diff --check`, commit final verification metadata, and prepare the independent external repository path without publishing.**

Commit: `chore(release): complete focused product verification`

## Self-review result

- Every requirement in the focused design maps to a task.
- The plan deletes broad provider and publication surfaces only after new tests define the focused behavior.
- Discovery revision, profile, release type, and Skill names are consistent across tasks.
- No public name, account, signing identity, or publication is inferred.
