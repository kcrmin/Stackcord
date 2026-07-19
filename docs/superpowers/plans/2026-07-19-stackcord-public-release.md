# Stackcord Public Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish version `1.0.0` of the existing product as the canonical `kcrmin/Stackcord` Codex Plugin and cross-platform CLI release.

**Architecture:** Rename every public identity boundary to Stackcord while preserving product behavior and Git history. Deterministic local verification precedes remote `main`, the exact tag, GitHub Actions, release assets, and a fresh public installation check.

**Tech Stack:** Codex Plugin manifest and marketplace, Go 1.26, Python 3, Git, GitHub CLI, GoReleaser-compatible release assets, GitHub Actions

## Global Constraints

- Plugin and marketplace identifier: `stackcord`.
- Display name: `Stackcord`.
- Repository: `https://github.com/kcrmin/Stackcord`.
- Version and tag: `1.0.0` and `v1.0.0`.
- Public CLI and asset stem: `stackcord`.
- Preserve implementation history and never force push.
- Do not publish if deterministic local verification or GitHub Actions fails.

---

### Task 1: Lock the production identity with tests

**Files:**
- Modify: `scripts/validate_plugin_test.py`
- Modify: `scripts/render_plugin_packages_test.py`
- Modify: `scripts/validate_release_config_test.py`
- Modify: `scripts/validate_docs_test.py`

**Interfaces:**
- Consumes: manifest, marketplace, package renderer, public docs, and release configuration
- Produces: failing tests for the `stackcord`, repository URL, CLI asset, and install-command contracts

- [ ] **Step 1: Add assertions for Stackcord public identity**

Require the manifest and marketplace to use `stackcord`, public URLs to use `kcrmin/Stackcord`, package names and platform assets to start with `stackcord`, and READMEs to document `kcrmin/Stackcord` plus `stackcord@stackcord`.

- [ ] **Step 2: Run the four focused test modules**

Run with `PYTHONPATH=scripts python3 -m unittest` and the four module names.

Expected: fail against the existing `fullstack-orchestrator` identity.

### Task 2: Rename the Plugin and public CLI

**Files:**
- Modify: `.codex-plugin/plugin.json`
- Modify: `.agents/plugins/marketplace.json`
- Modify: `cli/go.mod` and Go import paths
- Rename: `cli/cmd/orchestrator` to `cli/cmd/stackcord`
- Modify: `.goreleaser.yaml`
- Modify: `.github/workflows/ci.yml`
- Modify: release renderer, bootstrap scripts, validators, dogfood, and their tests

**Interfaces:**
- Consumes: identity tests from Task 1
- Produces: Plugin `stackcord`, module `github.com/kcrmin/Stackcord/cli`, command `stackcord`, and four platform assets

- [ ] **Step 1: Apply the mechanical identity migration**

Replace public product identifiers, module imports, command paths, upload names, and strict-profile URLs. Preserve schema meaning and CLI command behavior.

- [ ] **Step 2: Format and run focused tests**

Run `gofmt` on Go files, the four Python identity test modules, `go test ./...`, and `go vet ./...` from `cli/`.

Expected: all pass with no legacy public identity in shipped files.

### Task 3: Update installation and release documentation

**Files:**
- Modify: `README.md`
- Modify: `README.ko.md`
- Modify: `docs/getting-started/en.md`
- Modify: `docs/getting-started/ko.md`
- Modify: strict release packaging documents and templates

**Interfaces:**
- Consumes: the final manifest, marketplace, binary, and release asset names
- Produces: copy-pasteable Git marketplace and Plugin installation instructions

- [ ] **Step 1: Replace placeholder and legacy installation commands**

Use `codex plugin marketplace add kcrmin/Stackcord --ref v1.0.0` followed by `codex plugin add stackcord@stackcord` and link to the canonical repository.

- [ ] **Step 2: Run docs, Plugin, release-config, security, and package validators**

Expected: every validator exits successfully and the rendered Plugin zip validates after extraction.

### Task 4: Verify and publish the exact release

**Files:**
- Delete after execution: `docs/superpowers/plans/2026-07-19-stackcord-public-release.md`
- Create outside Git: platform binaries, Plugin zip packages, checksum manifest, and release notes

**Interfaces:**
- Consumes: one clean verified commit
- Produces: remote `main`, tag `v1.0.0`, GitHub Release, and publicly installable Plugin

- [ ] **Step 1: Remove this completed plan and commit the release source**

Verify the worktree is clean after the commit and that the remote still has no refs.

- [ ] **Step 2: Run final deterministic verification**

Run full Go tests, race on concurrent packages, all Python tests, dogfood, four cross-builds, package rendering, extracted-package validation, schema/docs/secret/release validators, and `git diff --check`.

- [ ] **Step 3: Push the exact commit to remote main and wait for Actions**

Add origin `https://github.com/kcrmin/Stackcord.git`, push `HEAD:main`, and wait for every workflow on that commit. Stop on any failure.

- [ ] **Step 4: Tag and publish version 1.0.0**

Create annotated tag `v1.0.0`, push it, publish all verified assets with checksums, and confirm release/tag/main commit equality.

- [ ] **Step 5: Verify public installation**

Use a temporary Codex home to add `kcrmin/Stackcord` at `v1.0.0`, install `stackcord@stackcord`, inspect the installed manifest, and confirm the source came from the public tag.
