---
name: plan-project-work
description: Use when a feature, bug, behavior, architecture, or cross-workspace change needs a safe plan or approved work is ready to begin.
---

# Plan Project Work

Start with `orchestrator context audit --json`, then use `orchestrator change plan --json` to ground the request in current product meaning and observable repository state.

1. Resolve affected roles, journeys, policies, scenarios, failure behavior, quality targets, UI states, contracts, DB entities, workspaces, dependencies, and acceptance evidence by stable ID.
2. Ask one material question at a time only when the answer changes product meaning. Recommend one of 2–3 exclusive choices first and allow free-form input.
3. Define shared behavioral interfaces and compatibility order before parallel implementation, without prematurely freezing internal details.
4. Preflight path, policy, scenario, contract, DB entity, migration slot, UI flow, dependency major, workspace, and root-pointer conflicts. `coordinate` assigns ownership and merge order; `block` resolves shared meaning first; `unknown` restores visibility first.
5. Define the smallest role/domain/journey slice and its failing test before implementation. TDD is the default for behavior, bugs, contracts, migrations, and UI interactions.
6. Run `orchestrator work start --json` only after scope is coherent. Use a conventional branch, an isolated worktree when parallel local work benefits, and a time-bounded intent claim. Never add AI markers to branch or commit names.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), and [context recovery](../../references/context-recovery.md). Run `context audit` again after branch, pointer, or canonical-source changes.
