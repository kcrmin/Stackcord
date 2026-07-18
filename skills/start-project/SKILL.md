---
name: start-project
description: Use when a service idea needs discovery, a framework-neutral full-stack project must be created, or an existing repository needs non-destructive adoption.
---

# Start Project

Build durable product understanding before selecting implementation technology. Let conversation stay natural; use the CLI only for observable state and safe writes.

1. Run `orchestrator status --json`. If no harness exists, treat that result as the starting diagnosis. If the CLI is absent, use the repo-local fallback and disclose reduced verification.
2. Inspect files, Git, existing configuration, and detected tools before asking anything discoverable.
3. Ask one decision only when its answer materially changes the product. Offer 2–3 exclusive choices, recommend the first, and accept free-form input.
4. After every material answer, normalize the complete current meaning and run `orchestrator project checkpoint`. Save product summary, goals and non-goals, roles, journeys, capabilities, service policies, failure behavior, scenarios, quality, UI coverage, data needs, decisions, assumptions, and open questions. Never save raw dialogue or tone.
5. Surface important unconsidered security, privacy, abuse, accessibility, failure, operations, observability, support, and data-lifecycle cases at the point they can change a decision.
6. Keep technology undecided until product, quality, team, and operating constraints justify it. At selection time, compare 2–3 suitable candidates using current official maintenance, security, release, license, platform, cost, export, and lock-in evidence. Connect only the chosen external tool.
7. Use `orchestrator project init` for a new root or `orchestrator project adopt` for an existing repository. Review the plan before apply, preserve user files, then rerun combined status.

Example: for “결제 실패를 줄이는 서비스를 만들고 싶어,” inspect first, checkpoint known intent, then ask the single highest-impact unanswered product question—not the framework.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), and [context recovery](../../references/context-recovery.md).
