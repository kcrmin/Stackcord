# Tiered Test Policy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebalance deterministic CI, release, agent evaluation, and Plugin packaging without reducing core product safety.

**Architecture:** Repository contract tests define the workflow and package boundaries. Pull requests use two representative native full-test jobs plus one fast contract job; release owns dogfood, race, four artifacts, checksums, and real Plugin installation. Agent model evaluation and strict publication remain explicit opt-ins.

**Tech Stack:** GitHub Actions YAML, Python 3 standard library, Go 1.26, Codex CLI, GoReleaser.

## Global Constraints

- Never invoke actual Codex from an ordinary pull request or normal release.
- Keep all nine behavior scenarios and the rubric committed.
- Require `--all` for a nine-scenario run and a separate opt-in for external-tool research.
- Preserve macOS amd64/arm64 and Windows amd64/arm64 release binaries.
- Do not include development tests or fixtures in Plugin zip files.
- Do not push, tag, publish, or create an external release.

---

### Task 1: Executable policy contracts

**Files:**
- Modify: `scripts/validate_ci_test.py`
- Modify: `scripts/validate_agent_eval_test.py`
- Modify: `scripts/render_plugin_packages_test.py`
- Modify: `scripts/validate_release_config_test.py`

**Interfaces:**
- Consumes: workflow YAML, `run_agent_eval.py`, rendered zip file lists.
- Produces: failing tests for PR/release/model/strict/package boundaries.

- [ ] Add tests that reject model execution in ordinary workflows, repeated four-platform full tests, dogfood in PR, test files in Plugin zip, and implicit all-scenario execution.
- [ ] Run the four focused Python modules and confirm the new assertions fail for the expected policy violations.

### Task 2: Agent evaluation opt-in

**Files:**
- Modify: `scripts/run_agent_eval.py`
- Modify: `scripts/validate_agent_eval.py`
- Create: `.github/workflows/skill-eval.yml`

**Interfaces:**
- Consumes: one `--scenario`, or explicit `--all`; optional explicit external-research permission.
- Produces: selected Codex drill results under ignored `.harness/local/evals/`.

- [ ] Require one selected scenario by default and require `--all` for the complete suite.
- [ ] Reject `current-tool-selection` unless `--allow-external-research` is present.
- [ ] Add a manual workflow that accepts exactly one non-research scenario.
- [ ] Run focused tests until green without invoking Codex.

### Task 3: Tiered PR and release workflows

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/security.yml`
- Modify: `.github/workflows/release.yml`
- Create: `.github/workflows/strict-release.yml`

**Interfaces:**
- Consumes: source tree and explicit release inputs.
- Produces: representative PR evidence, complete deterministic release evidence, optional strict evidence.

- [ ] Keep full tests on macOS ARM and Windows x64; make macOS Intel and Windows ARM build-smoke release targets.
- [ ] Move dogfood, race, and four-target packaging to normal release.
- [ ] Remove duplicate PR secret scanning and keep vulnerability/security coverage.
- [ ] Isolate strict tests and guards behind path selection or explicit dispatch.
- [ ] Run CI and release configuration validators until green.

### Task 4: Lean Plugin package and real installation smoke

**Files:**
- Modify: `scripts/render_plugin_packages.py`
- Create: `scripts/smoke_install_plugin.py`
- Modify: `scripts/render_plugin_packages_test.py`
- Modify: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: source Plugin root or rendered Plugin zip, isolated `CODEX_HOME`, installed Codex CLI.
- Produces: verified source/rendered local marketplace installation without publication.

- [ ] Define an allowlist that excludes Go, eval, dogfood, testdata, CI, and all Python test files.
- [ ] Add a dry deterministic command-construction test and a real release smoke entry point.
- [ ] Verify both source and one rendered package using actual Codex plugin commands during release.
- [ ] Run package and Plugin validators until green.

### Task 5: User-facing policy and historical evidence

**Files:**
- Modify: `README.md`
- Modify: `README.ko.md`
- Modify: `docs/release-readiness.md`
- Create: `docs/guides/testing-en.md`
- Create: `docs/guides/testing-ko.md`
- Modify: `scripts/validate_docs.py`

**Interfaces:**
- Consumes: final workflow and command boundaries.
- Produces: concise bilingual explanation that separates automatic checks from historical model evidence.

- [ ] Document the five execution tiers and opt-in commands.
- [ ] Record the prior 9/9 model run as historical local evidence, never current CI evidence.
- [ ] Add testing guide parity validation and run documentation checks.

### Task 6: Deterministic verification

**Files:**
- Verify only.

**Interfaces:**
- Consumes: completed repository.
- Produces: fresh deterministic evidence with zero Codex calls and no external publication.

- [ ] Run all Python tests and validators.
- [ ] Run complete Go tests and race tests once locally.
- [ ] Run multi-repository dogfood.
- [ ] Render packages and assert excluded paths are absent.
- [ ] Build all four supported binaries.
- [ ] Confirm clean Git diff checks and no workflow invokes `run_agent_eval.py` except the manual Skill workflow.

