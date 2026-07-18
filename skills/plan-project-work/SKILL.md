---
name: plan-project-work
description: Use when a feature, bug, behavior, architecture, or cross-workspace change needs an executable checklist, conflict-aware ownership, or a safe start.
---

# Plan Project Work

Turn a product request into the smallest testable change. Use durable coordination only when it adds recovery or collision-prevention value; a small private local change does not need a ticket or work reservation.

1. Run `orchestrator status --json`, then `orchestrator change plan --json`. Resolve affected roles, journeys, service policies, failure scenarios, quality targets, UI states, contracts, DB entities, migrations, workspaces, dependencies, and acceptance evidence.
2. Ask one material question only when its answer changes product meaning. Put the recommended option first among 2–3 exclusive choices and accept free-form input.
3. Classify coordination need from observable facts. A short-lived private edit in one workspace with no shared semantic boundary may proceed with the ordinary diff, test, and review. Shared ownership, interruption recovery, cross-workspace work, policy or contract changes, migrations, shared UI flows, dependency upgrades, and likely parallel overlap use the remaining coordination steps.
4. For coordinated work, define shared behavioral interfaces and compatibility order, then run `orchestrator work conflict --json` for path, policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer overlap. Restore unknown visibility, assign owned overlap and merge order, and settle blocked shared meaning before implementation. Do not freeze unrelated internals.
5. Use `orchestrator work define --json` to save the coordinated executable checklist with stable acceptance and failure references, ordered workspace slices, semantic scope, dependencies, merge order, first failing test, and required evidence. The selected task provider owns live status; the repository owns durable product meaning and the implementation boundary.
6. Re-read the selected provider immediately before writing. Use Git-local when no external provider is selected. For GitHub, Jira, Beads, or another selected source, use its real installed connector or authenticated CLI to create or update the visible item, assign the intended owner, and move it to `in_progress`. Normalize and reconcile the exact item revision with `orchestrator work provider reconcile --apply`. Never fabricate an issue, assignee, or provider update.
7. Define the first failing test. TDD is required for behavior, bugs, contracts, migrations, and UI interactions; a documentation-only correction uses a focused diff and applicable validation instead of inventing a test.
8. For coordinated work, run `orchestrator work start --apply` after the external assignment is confirmed. The command performs the Git compare-and-swap semantic reservation; if another contributor wins or meaning overlaps, it creates no branch state. Re-read both sources after success. Use a conventional branch such as `feature/refund-failure` or `fix/login-race`; include a real issue key only when the team convention uses it. Never include AI, agent, model, or tool names.
9. Use `orchestrator git worktree --apply` only when isolated parallel work reduces collision. Report the ordinary task name and workspace, not internal reservation IDs or files.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), [context recovery](../../references/context-recovery.md), and [tool selection](../../references/tool-selection.md). Rerun combined status after branch, worktree, provider, pointer, or canonical-source changes.
