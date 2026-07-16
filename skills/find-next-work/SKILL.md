---
name: find-next-work
description: Use when a user asks what to do next, which task is ready, why work is blocked, or how to continue toward a complete release.
---

# Find Next Work

Recommend executable work from dependencies and evidence, not task-title intuition.

1. Run `orchestrator context audit --json`, then `orchestrator work next --json`.
2. Use the selected single live task-status provider; Git-local is the offline fallback. Product meaning remains in specs and contracts.
3. Exclude work whose policy, scenario, contract, migration, UI flow, dependency, workspace, or release gate is stale, blocked, claimed, or unknown.
4. Rank ready work by lifecycle dependency, user value, unblock impact, risk, and smallest reviewable vertical slice.
5. Return the recommendation, prerequisites, acceptance references, TDD starting test, conflict scope, and why alternatives wait. Ask only when equally valid choices change product intent.

Read [lifecycle](../../references/lifecycle.md) and [context recovery](../../references/context-recovery.md).
