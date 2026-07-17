---
name: continue-project
description: Use when a cloned, transferred, or paused project must be resumed, the next safe work is unclear, or live progress and blockers need reconstruction.
---

# Continue Project

Start with `orchestrator context audit --json` and summarize product state in plain language. Do not rely on prior chat or another contributor's memory.

1. Inspect actual root, branch, dirty state, upstream, ahead/behind/divergence, worktrees, workspace commits, exact submodule pointers, selected task source, active claims, specs, contracts, stale items, and evidence without mutation.
2. Separate confirmed facts, stale derivations, unknown external state, and blockers. Never auto-pull, rebase, stash, reset, initialize submodules, or move pointers.
3. If context is coherent, run `orchestrator work next --json`. Exclude work with unresolved policy, scenario, contract, migration, UI, dependency, ownership, or observability conflicts.
4. Recommend one small reviewable change by user value, unblock impact, risk, and dependency order. Include acceptance references, the first failing test, affected workspace, and why alternatives wait.
5. Ask only when equally valid choices change product intent or unsafe Git reconciliation needs user direction.

Read [context recovery](../../references/context-recovery.md), [workflow](../../references/workflow.md), and [safety](../../references/safety.md). If the CLI is missing, use the repo-local fallback and explicitly state reduced verification.
