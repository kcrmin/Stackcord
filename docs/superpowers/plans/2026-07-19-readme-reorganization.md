# README Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite both public READMEs around the nine problems Stackcord solves while removing repeated feature explanations.

**Architecture:** `README.ko.md` is the copy and information-architecture baseline. `README.md` mirrors its sections, claims, examples, tables, and links in English; detailed operational material stays in the existing guides.

**Tech Stack:** GitHub-flavored Markdown, Mermaid, repository Python validators

## Global Constraints

- Target a recently graduated developer.
- Preserve all nine established product problems and shipped capabilities.
- Put natural-language Codex usage before CLI and internal paths.
- Describe provider enforcement limits accurately.
- Keep Korean and English semantically aligned.
- Target 130–160 lines per README.

---

### Task 1: Rewrite the Korean README

**Files:**
- Modify: `README.ko.md`

**Interfaces:**
- Consumes: shipped behavior documented by current README and `docs/superpowers/specs/2026-07-19-readme-information-architecture-design.md`
- Produces: the canonical section order and Korean copy mirrored by Task 2

- [ ] **Step 1: Replace the repeated information architecture**

Keep the nine-row problem table, then use exactly three short examples for discovery, tool recommendation, and product-authority review. Follow with one workflow, one collaboration structure, one verification-boundary table, installation, generated files, and guide links.

- [ ] **Step 2: Check scope and readability**

Run: `wc -l README.ko.md`

Expected: 130–160 lines, with no removal of QDD, framework neutrality, full-stack submodules, context recovery, semantic conflicts, external tools, TDD, or exact release candidates.

- [ ] **Step 3: Commit the Korean baseline**

```bash
git add README.ko.md
git commit -m "docs: simplify Korean README"
```

### Task 2: Mirror the English README

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: the completed heading, table, and example structure from `README.ko.md`
- Produces: semantically equivalent English public documentation

- [ ] **Step 1: Rewrite the English copy to match the Korean baseline**

Mirror every heading, problem row, dialogue, workflow stage, verification boundary, generated path, and guide link. Use natural English rather than sentence-by-sentence literal translation.

- [ ] **Step 2: Check structural parity**

Run: `wc -l README.ko.md README.md`

Expected: both files are 130–160 lines and have matching section/table structure.

- [ ] **Step 3: Commit the English mirror**

```bash
git add README.md
git commit -m "docs: simplify English README"
```

### Task 3: Validate the public documentation

**Files:**
- Test: `scripts/validate_docs.py`
- Test: `scripts/validate_plugin.py`

**Interfaces:**
- Consumes: both rewritten READMEs and their local links
- Produces: evidence that documentation parity and packaged Plugin claims remain valid

- [ ] **Step 1: Run documentation and Plugin validation**

```bash
python3 scripts/validate_docs.py .
python3 scripts/validate_plugin.py .
```

Expected: documentation parity passes for all English/Korean pairs and Plugin validation passes.

- [ ] **Step 2: Check Markdown whitespace and repository state**

```bash
git diff --check
git status --short --branch
```

Expected: no whitespace errors and only the implementation-plan cleanup, if any, remains uncommitted.

- [ ] **Step 3: Remove the completed implementation plan if required by repository validation and commit**

The repository treats active plan files as temporary execution state. Remove this plan after all tasks pass, then commit the removal without altering the retained design specification.

```bash
git add docs/superpowers/plans/2026-07-19-readme-reorganization.md
git commit -m "docs: complete README reorganization plan"
```
