---
name: plan-project-work
description: Use when a feature, bug, behavior, architecture, or cross-workspace change needs an executable checklist, conflict-aware ownership, or a safe start.
---

# Plan Project Work

Turn a product request into the smallest testable cross-workspace change, then reserve it before creating implementation state. In user-facing replies, call the internal claim mechanism “작업 선점” or “work reservation.”

1. Run `orchestrator status --json`, then `orchestrator change plan --json`. Resolve affected roles, journeys, service policies, failure scenarios, quality targets, UI states, contracts, DB entities, migrations, workspaces, dependencies, and acceptance evidence.
2. Ask one material question only when its answer changes product meaning. Put the recommended option first among 2–3 exclusive choices and accept free-form input.
3. Define shared behavioral interfaces and compatibility order before parallel work; do not freeze unrelated internals.
4. Run `orchestrator work conflict --json` for path, policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer overlap. Resolve `unknown` visibility first, coordinate owned overlap and merge order, and settle blocked shared meaning before implementation.
5. Use `orchestrator work define --json` to save an executable checklist with stable acceptance references, ordered workspace slices, dependencies, risks, and required evidence. The selected task provider owns live status; the repository owns durable product meaning.
6. Re-read the selected provider immediately before writing. Use Git-local by default when no external provider is selected; use GitHub, Jira, or Beads only through its real installed connector or authenticated CLI. Never fabricate an issue, assignee, or claim.
7. Define the first failing test. TDD is required for behavior, bugs, contracts, migrations, and UI interactions.
8. Run `orchestrator work start --json` only after the remote reservation succeeds, then re-read the provider. Create a conventional branch such as `feature/refund-failure` or `fix/login-race`; include a real issue key only when the team convention uses it. Never include AI, agent, model, or tool names.
9. Use `orchestrator git worktree-plan` and `git worktree` when isolated parallel work reduces collision. Report the ordinary task name and workspace, not internal reservation IDs or files.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), and [context recovery](../../references/context-recovery.md). Rerun combined status after branch, worktree, provider, pointer, or canonical-source changes.
