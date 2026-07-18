# Editable UI Baseline Workspace Implementation Plan

> Execute sequentially with focused red-green-refactor TDD. Run the full, race, security, Plugin, and cross-platform suites once at the end instead of repeating them after every task.

**Goal:** Add an optional editable `ui/` workspace/submodule, safely bring external UI material into it, bind frontend work to an exact UI baseline, and simplify the bilingual product entry experience.

**Architecture:** Skills handle product conversation, UI choices, and optional tools. The Go CLI handles only deterministic Git mutation, workspace identity, import safety, promotion, baseline identity, staleness, and release verification. An external source record preserves provenance; a separate UI baseline binds the edited UI repository commit to product flows and frontend consumers.

**Design:** `docs/superpowers/specs/2026-07-18-ui-baseline-submodule-design.md`

## Test budget

Add tests only where a regression could corrupt repositories, lose provenance, miss stale frontend work, or break clone recovery.

- one focused Go test group per new boundary;
- one representative root + UI + frontend + backend E2E;
- small deterministic Skill and bilingual-document validators;
- one final full suite, race suite, security scan, Plugin validation, and four cross-builds;
- no combinatorial provider, framework, design-tool, or archive-format matrix.

## Task 1: UI baseline identity

**Files:**

- Create `schemas/ui-baseline.schema.json`
- Create `cli/internal/schema/definitions/ui-baseline.schema.json`
- Create `cli/internal/ui/baseline.go`
- Create `cli/internal/ui/baseline_test.go`
- Modify `cli/internal/schema/loader.go`
- Modify `cli/internal/work/model.go`
- Modify `cli/internal/work/definition.go`
- Modify both work-item schemas and `cli/internal/work/definition_test.go`

**Focused test:** one table test covers valid baseline, unsafe identity, deterministic fingerprint, and a frontend work definition using a stale fingerprint.

**Implementation:** store `ui.baseline.*`, workspace ID, exact commit/remote, optional source IDs, mapped UI refs, consumers, and digest. Add `ui_baselines` to executable work so its normal definition fingerprint also binds the UI baseline.

**Commit:** `feat(ui): define versioned UI baselines`

## Task 2: Safe submodule creation and workspace registration

**Files:**

- Create `cli/internal/gitx/submodule_add.go` and `_test.go`
- Modify `cli/internal/gitx/mutate.go`, `runner.go`, `cli/internal/command/git.go`
- Create `cli/internal/workspace/register.go` and `_test.go`
- Create `cli/internal/command/workspace.go`
- Create framework-neutral files under `templates/workspace/ui/`
- Modify `cli/internal/command/root.go`, workspace loader/schema/tests

**Focused tests:**

1. safe URL/path and exact allowlisted submodule command with postconditions;
2. non-destructive UI workspace registration, child bridge, frontend dependency, and cycle/duplicate rejection.

**Commands:**

```text
orchestrator git submodule add --root . --remote <url> --path ui [--apply]
orchestrator workspace register --root . --id workspace.ui --kind submodule \
  --path ui --responsibility ui-baseline --consumer workspace.frontend \
  --initialize ui [--apply]
```

Neither command creates a remote, commits, pushes, or overwrites user work implicitly.

**Commit:** `feat(workspace): add editable UI boundaries`

## Task 3: External UI promotion and exact baseline binding

**Files:**

- Create `cli/internal/ui/promote.go` and `_test.go`
- Modify `cli/internal/ui/import.go`, `reconcile.go`, `baseline.go`
- Modify `cli/internal/command/boundaries.go`
- Modify external-source schemas as needed
- Modify context, continuity, work, and integration validation

**Focused tests:**

1. `whole`, `selected`, and `reference-only` promotion from an unchanged inspected source, including escape/symlink/overwrite rejection;
2. binding a clean published UI commit and reporting dirty, local-only, remote, root-pointer, or stale-frontend mismatch.

**Commands:**

```text
orchestrator ui promote --root . --id ui.external.checkout \
  --workspace workspace.ui --mode whole [--path <relative>] [--apply]

orchestrator ui baseline bind --root . --id ui.baseline.checkout \
  --workspace workspace.ui --source ui.external.checkout \
  --ref ui.checkout --consumer workspace.frontend [--apply]
```

Promoted files remain ordinary editable files. The CLI does not invent screen structure or choose a visual direction. It writes a root-owned baseline record and tells the user when the UI gitlink and baseline record must be committed together.

**Commit:** `feat(ui): connect editable baselines to frontend work`

## Task 4: Skills, generated fallback, and concise bilingual documentation

**Files:**

- Modify existing five `skills/*/SKILL.md` files only where their role changes
- Modify shared `references/workflow.md`, `safety.md`, and `tool-selection.md`
- Modify behavior fixtures/evals before Skill text
- Modify generated repo-local Skill/fallback templates
- Create `docs/guides/ui-workspace-ko.md` and `ui-workspace-en.md`
- Rewrite `README.ko.md` and `README.md`
- Update getting-started, submodule, and troubleshooting guides

**Focused tests:** one behavior contract and one bilingual documentation structure/parity test.

README contains one sentence, four effects, three natural prompts, one small Mermaid flow, a compact feature table, a five-minute path, default/strict distinction, and links to details. The UI guide explains:

- A: no baseline — create and edit one;
- B: partial baseline — compare and selectively merge;
- C: approved external baseline — import, edit, and complete missing states.

MengTo/Skills-like tools are detected or suggested only when useful. They are optional inputs, not service truth or release identity.

**Commit:** `docs(product): explain editable UI continuity`

## Task 5: Representative production E2E

**Files:**

- Modify `cli/internal/command/production_e2e_test.go`
- Modify `dogfood/scenario.yaml`, `run.py`, expected results, report, and README

**One E2E:** create temporary root/UI/frontend/backend repositories, add/register UI, import and promote one mockup, edit and publish UI, bind the baseline, define frontend work, record TDD/integration evidence, detect a newer stale UI baseline, then recover the same state from a fresh root clone and child clone.

Only defects revealed by this E2E may add glue or another narrow regression test.

**Commit:** `test(product): prove UI workspace continuity`

## Task 6: Final verification and release readiness

Run once from the final implementation:

```bash
gofmt -w $(find cli -name '*.go' -type f)
git diff --check
cd cli && go test ./... && go test -race ./... && cd ..
python3 -m unittest discover -s scripts -p '*_test.py'
python3 scripts/validate_plugin.py .
python3 scripts/validate_docs.py .
python3 scripts/security_scan.py .
python3 dogfood/run.py
bash scripts/validate-plugin.sh
```

Build:

```bash
cd cli
GOOS=darwin GOARCH=amd64 go build -o ../dist/cross/orchestrator_darwin_amd64 ./cmd/orchestrator
GOOS=darwin GOARCH=arm64 go build -o ../dist/cross/orchestrator_darwin_arm64 ./cmd/orchestrator
GOOS=windows GOARCH=amd64 go build -o ../dist/cross/orchestrator_windows_amd64.exe ./cmd/orchestrator
GOOS=windows GOARCH=arm64 go build -o ../dist/cross/orchestrator_windows_arm64.exe ./cmd/orchestrator
```

Update `docs/release-readiness.md` with actual results and limitations. Do not publish.

**Commit:** `docs(release): verify editable UI readiness`

## Completion conditions

- clean worktree and final verification evidence;
- editable imported material and exact UI baseline identity;
- framework-neutral and optional submodule topology;
- stale UI/frontend and root-pointer state detected after clone;
- Plugin-less continuation remains functional;
- optional UI tools never become source of truth;
- conventional branch and commit names contain no AI branding;
- no implicit remote creation, commit, push, PR, account, package, or release.
