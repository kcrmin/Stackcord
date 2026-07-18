---
name: start-project
description: Use when a service idea needs discovery, a framework-neutral full-stack project must be created, or an existing repository needs non-destructive adoption.
---

# Start Project

Build durable product understanding before selecting implementation technology. Let conversation stay natural; use the CLI only for observable state and safe writes.

1. Resolve the CLI from `ORCHESTRATOR_CLI`, a repository build, or `PATH`, then run `orchestrator status --json`. If no harness exists, treat that result as the starting diagnosis. On the first explicit product request only, an absent CLI may trigger an offer to run the matching verified `scripts/bootstrap-cli.sh` or `scripts/bootstrap-cli.ps1` with an explicit release URL, version, and install directory. Hooks never install software. If installation is unavailable or declined, use the repo-local fallback and disclose reduced verification.
2. Inspect files, Git, existing configuration, and detected tools before asking anything discoverable.
3. Treat the initial product request as the first material answer. Before asking the next question, and after every later material answer, normalize the complete current meaning. Use the valid input shape printed by `orchestrator project checkpoint --help`, review the plan, apply it, and verify the successful apply before continuing. Save product summary, goals and non-goals, roles, journeys, capabilities, service policies, failure behavior, scenarios, quality, UI coverage, data needs, decisions, assumptions, and open questions. Never save raw dialogue or tone.
4. Ask one decision only when its answer materially changes the product. When choices help, present 2–3 exclusive options labeled A/B/C, put the recommended option first and mark it recommended, then accept either a letter or free-form input.
5. Surface important unconsidered security, privacy, abuse, accessibility, failure, operations, observability, support, and data-lifecycle cases at the point they can change a decision.
6. Keep technology undecided until product, quality, team, and operating constraints justify it. When a framework, task provider, design tool, database visualizer, CI system, or complementary workflow becomes useful, inspect installed tools first, search current official evidence when the choice can change, compare 2–3 realistic candidates, and explain their source-of-truth boundary. Connect only the user's choice; Superpowers, BMAD, Beads, Memory tools, GitHub, Jira, and future tools are optional complements rather than hidden dependencies.
7. Use `orchestrator project init` for a new root or `orchestrator project adopt` for an existing repository. Review the plan before apply, preserve user files, then rerun combined status. Initialize collaboration Git early, but add a child repository or submodule only when its ownership or lifecycle is genuinely independent.

Example: for “결제 실패를 줄이는 서비스를 만들고 싶어,” inspect first, checkpoint known intent, then ask the single highest-impact unanswered product question—not the framework.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), [context recovery](../../references/context-recovery.md), and [tool selection](../../references/tool-selection.md).
