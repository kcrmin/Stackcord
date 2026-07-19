# Product Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add provider-account-based product authority so contributors can propose protected product changes while unapproved meaning is blocked from integration and release.

**Architecture:** A focused `governance` package loads one committed policy and one ignored normalized live review observation. It fingerprints protected product sources, verifies exact provider account decisions, exposes a read-only CLI check, and contributes immutable identity to combined status, integration plans, and release candidates. Git provider access controls enforce merge permissions; Stackcord verifies semantic scope and exact approval without inventing provider adapters.

**Tech Stack:** Go, Cobra, JSON Schema 2020-12, YAML, Git, Markdown, Python repository validators.

## Global Constraints

- Git `user.name` and `user.email` never prove authority.
- Existing projects without enabled governance retain their current workflow.
- Provider observations are ignored local evidence and must be fresh, normalized, secret-free, and exact-commit-bound.
- Ordinary contributors may propose protected changes; only configured provider subjects can approve them.
- No live provider or Codex call runs in deterministic tests.
- Core mode stays lightweight; cryptographic multi-approval remains an optional strict extension.
- User-facing text hides `.harness/`, observation IDs, and internal approval mechanics unless troubleshooting requires them.

---

## File map

- `cli/internal/governance/model.go`: policy, authority, decision, observation, and report types.
- `cli/internal/governance/load.go`: strict safe loading and local-observation path validation.
- `cli/internal/governance/fingerprint.go`: deterministic protected-meaning fingerprint.
- `cli/internal/governance/verify.go`: disabled, proposed, approved, stale, and unknown decisions.
- `cli/internal/governance/*_test.go`: deterministic policy, spoofing, freshness, and exact-identity tests.
- `schemas/governance*.schema.json` and `cli/internal/schema/definitions/governance*.schema.json`: public and embedded schemas.
- `cli/internal/command/governance.go`: `orchestrator governance check` command.
- `cli/internal/continuity/*`: combined governance view and next action.
- `cli/internal/integration/model.go`, `cli/internal/command/integrate.go`: exact approval identity in integration plans.
- `cli/internal/release/*`, `schemas/release-candidate.schema.json`: exact approval identity in release candidates.
- `cli/internal/project/generate.go`, `templates/project/**`: disabled-by-default governance scaffold and repo-local guidance.
- `skills/coordinate-project-work/SKILL.md`, `skills/recover-and-release-project/SKILL.md`, `references/safety.md`: AI proposal and approval behavior.
- `docs/guides/governance-{ko,en}.md`, `README.ko.md`, `README.md`: concise user documentation and problem statement.

### Task 1: Deterministic governance verifier

**Files:**
- Create: `cli/internal/governance/model.go`
- Create: `cli/internal/governance/load.go`
- Create: `cli/internal/governance/fingerprint.go`
- Create: `cli/internal/governance/verify.go`
- Create: `cli/internal/governance/governance_test.go`
- Create: `schemas/governance.schema.json`
- Create: `schemas/governance-observation.schema.json`
- Create: `cli/internal/schema/definitions/governance.schema.json`
- Create: `cli/internal/schema/definitions/governance-observation.schema.json`

**Interfaces:**
- Produces: `governance.Check(ctx, root, observationPath, now) Report`.
- Produces: `Report{Enabled, Status, ProtectedFingerprint, ApprovalRevision, Authorities, Approvers, Issues}`.

- [ ] Write tests proving disabled policy passes, enabled policy needs a fresh exact observation, only configured subjects count, duplicate subjects count once, self-approval follows policy, stale commit/fingerprint/provider/repository/revision is rejected, Git display identity is absent from the model, unsafe files and paths are rejected, and a governance change is included in the protected fingerprint.
- [ ] Run `go test ./internal/governance` and confirm the package is missing or tests fail for the intended behavior.
- [ ] Add strict models, duplicated public/embedded schemas, safe loaders, deterministic tree hashing, and exact verification with a 15-minute freshness limit.
- [ ] Run `go test ./internal/governance ./internal/schema` and confirm all tests pass.
- [ ] Run `gofmt -w internal/governance` and repeat the tests.

### Task 2: CLI, status, integration, and release gates

**Files:**
- Create: `cli/internal/command/governance.go`
- Create: `cli/internal/command/governance_test.go`
- Modify: `cli/internal/command/root.go`
- Modify: `cli/internal/command/root_test.go`
- Modify: `cli/internal/continuity/model.go`
- Modify: `cli/internal/continuity/collect.go`
- Modify: `cli/internal/continuity/next.go`
- Modify: `cli/internal/continuity/continuity_test.go`
- Modify: `cli/internal/integration/model.go`
- Modify: `cli/internal/command/integrate.go`
- Modify: `cli/internal/release/candidate.go`
- Modify: `cli/internal/release/gate.go`
- Modify: `cli/internal/release/collect.go`
- Modify: `cli/internal/release/release_test.go`
- Modify: `cli/internal/release/collect_test.go`
- Modify: `schemas/release-candidate.schema.json`
- Modify: `cli/internal/schema/definitions/release-candidate.schema.json`

**Interfaces:**
- Consumes: `governance.Check`.
- Produces: read-only `orchestrator governance check --root --observation --json`.
- Produces: `GovernanceFingerprint` and `GovernanceApprovalRevision` in integration and release identity.

- [ ] Write command, continuity, integration, and release tests showing disabled governance is non-blocking, enabled missing approval is unknown/blocked, approved exact evidence passes, changed governance identity invalidates a recorded integration plan and release candidate, and the new command appears in the public surface.
- [ ] Run the focused tests and confirm the new expectations fail.
- [ ] Add the CLI command and combined status view; map governance unknown or blocked state to one plain-language next action.
- [ ] Bind governance fingerprint and approval revision into integration plans and release candidate input, validation, clone, equality checks, and JSON schemas.
- [ ] Run `go test ./internal/command ./internal/continuity ./internal/integration ./internal/release ./internal/schema` until green.

### Task 3: Generated project and Skill behavior

**Files:**
- Modify: `cli/internal/project/generate.go`
- Modify: `cli/internal/project/project_e2e_test.go`
- Create: `templates/project/.harness/governance.yaml`
- Modify: `templates/project/AGENTS.md.tmpl`
- Modify: `templates/project/.agents/skills/use-project-harness/SKILL.md`
- Modify: `templates/project/.agents/skills/use-project-harness/references/fallback.md`
- Modify: `skills/coordinate-project-work/SKILL.md`
- Modify: `skills/recover-and-release-project/SKILL.md`
- Modify: `references/safety.md`

**Interfaces:**
- Consumes: governance policy and check result.
- Produces: disabled-by-default scaffold and natural-language proposal/approval behavior.

- [ ] Extend project E2E and generated-guidance assertions first: governance file exists, is disabled until a real provider identity is selected, general contributors propose, product authorities approve, local Git name/email never authorizes, and missing live evidence blocks protected integration/release.
- [ ] Run `go test ./internal/project` and confirm the assertions fail.
- [ ] Add the scaffold and concise guidance to generated, repo-local, fallback, and Plugin Skills without adding another public Skill.
- [ ] Run project tests and `python3 scripts/validate_plugin.py .` until green.

### Task 4: Bilingual README, guide, and release documentation

**Files:**
- Modify: `README.ko.md`
- Modify: `README.md`
- Create: `docs/guides/governance-ko.md`
- Create: `docs/guides/governance-en.md`
- Modify: `docs/release-readiness.md`
- Modify: `scripts/validate_docs_test.py`

**Interfaces:**
- Produces: junior-developer-level explanation of assignment, proposal, PR approval, clone recovery, limitations, and release blocking.

- [ ] Add failing documentation tests or validator expectations for the bilingual governance pair, the shared-product-understanding problem, the selected eight additional problem statements, and a readable grouped guide table instead of the seven-link inline footer.
- [ ] Run the focused Python documentation and Plugin validators and confirm the new expectation fails.
- [ ] Rewrite both README problem tables, add one short contributor/authority conversation, add the governance guide pair, and update release readiness without claiming local edit prevention or current automatic provider support.
- [ ] Run `python3 -m unittest scripts.validate_docs_test scripts.validate_plugin_test` and the direct validators until green.

### Task 5: Final deterministic verification

**Files:**
- Modify only defects found by verification.

- [ ] Run `go test ./...` from `cli/`.
- [ ] Run `go test -race ./internal/governance ./internal/provider ./internal/policy` from `cli/` because authorization and live-observation logic changed.
- [ ] Run `python3 -m unittest discover -s scripts -p '*_test.py'`.
- [ ] Run `python3 scripts/validate_plugin.py .`, `python3 scripts/validate_docs.py .`, and `python3 scripts/security_scan.py .`.
- [ ] Run the existing multi-repository dogfood command documented in `dogfood/README.md`.
- [ ] Run `git diff --check`, inspect the final diff, and confirm no Plugin package includes test fixtures or local observations.
