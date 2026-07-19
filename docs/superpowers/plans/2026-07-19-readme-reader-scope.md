# README Reader Scope Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove maintainer-level Git and verification inventories from both READMEs and explain the difference between product specifications and implementation contracts.

**Architecture:** The public README remains problem- and conversation-led. Detailed Git conventions and verification behavior stay in the existing task-management, submodule, governance, and troubleshooting guides, while the documentation validator checks each requirement at its appropriate reader level.

**Tech Stack:** GitHub-flavored Markdown, Python `unittest`, repository documentation validator

## Global Constraints

- Keep Korean and English semantically aligned.
- Preserve the nine product problems and three conversation examples.
- Replace the first example request with a natural expression of need.
- Remove the Git/submodule collaboration inventory and deterministic verification inventory from both READMEs.
- Explain `specs/` as product intent and `contracts/` as implementation obligations with one reservation example.

---

### Task 1: Define the reader-focused documentation contract

**Files:**
- Modify: `scripts/validate_docs_test.py`
- Modify: `scripts/validate_docs.py`

**Interfaces:**
- Consumes: README and task-management guide text
- Produces: validation that detailed Git conventions live in the guide while README keeps product-facing concepts

- [ ] **Step 1: Add a failing README-scope test**

Assert that both READMEs omit their Git/submodule and verification-inventory headings, contain `specs/` and `contracts/`, and include the reservation policy on both the intent and obligation sides.

- [ ] **Step 2: Run the focused test and observe failure**

Run: `python3 -m unittest scripts/validate_docs_test.py`

Expected: failure because the current README still contains the removed headings and lacks the explicit distinction.

- [ ] **Step 3: Move AI-free Git convention validation to task-management guides**

Update `public_contract_errors` so README no longer needs branch examples. Verify the existing English and Korean task-management guides contain the conventional branch, commit, and no-AI-branding tokens.

### Task 2: Simplify both READMEs

**Files:**
- Modify: `README.ko.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: the reader-scope test from Task 1
- Produces: aligned public READMEs with a concise specs/contracts explanation

- [ ] **Step 1: Update the discovery request and remove both detailed sections**

Use “예약 서비스도 필요할 것 같아” and “I think we also need a reservation service.” Remove the collaboration inventory, branch-convention paragraph, verification table, and provider-enforcement paragraph.

- [ ] **Step 2: Add the specs/contracts distinction**

After the lifecycle flow, explain that specs answer what/why and contracts define obligations implementations must obey. Use administrator approval as the shared example.

- [ ] **Step 3: Run focused and full documentation validation**

Run:

```bash
python3 -m unittest scripts/validate_docs_test.py
python3 scripts/validate_docs.py .
python3 scripts/validate_plugin.py .
git diff --check
```

Expected: all commands exit successfully.

- [ ] **Step 4: Commit the implementation and remove this completed plan**

Commit the validator and README change using a normal documentation commit, then delete this temporary plan and commit its removal. Retain the README design specification.
