---
name: resume-project
description: Use when a cloned, transferred, or previously paused project must be understood and continued without relying on earlier chat or contributor memory.
---

# Resume Project

Reconstruct current truth before recommending work.

1. Read [context recovery](../../references/context-recovery.md) and run `orchestrator context audit --json`.
2. Inspect actual root, branch, dirty files, remote tracking, worktrees, workspace commits, exact submodule pointers, active work, claims, contracts, baselines, and gates without mutation.
3. Distinguish facts, stale results, unknown external state, and blockers. Never auto-pull, rebase, stash, reset, or move a submodule pointer.
4. Run `orchestrator work next --json` only after context is coherent. Ask only if a material product choice or unsafe reconciliation remains.
5. State the current lifecycle gate, in-progress ownership, conflicts, evidence, and one safe next action.

Use [approval](../../references/approval.md) for any later mutation. If the CLI is missing, use the repo-local `use-project-harness` fallback.
