---
name: plan-project-change
description: Use when a feature, bug fix, behavior change, architecture change, or cross-workspace implementation request needs a safe executable plan.
---

# Plan Project Change

Turn the request into one coherent product and integration change before code.

1. Run `orchestrator context audit --json` and `orchestrator change plan --json`.
2. Resolve related role, journey, policy, scenario, failure behavior, quality target, UI state, contract, DB entity, workspace, and acceptance evidence by stable ID.
3. Ask one material question at a time only when the answer changes product meaning. Recommend a choice first and permit free-form other input.
4. Define shared interfaces and compatibility order before parallel implementation. Prefer additive/versioned boundaries, generated mocks/clients, and small vertical slices.
5. Run conflict preflight; assign paths and semantic scopes, TDD red test, dependency/merge/deploy order, rollback, and evidence.

Read [lifecycle](../../references/lifecycle.md), [approval](../../references/approval.md), and [context recovery](../../references/context-recovery.md).
