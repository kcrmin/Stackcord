---
name: start-project
description: Use when a service idea needs detailed discovery, a framework-neutral project must be created, or an existing repository needs non-destructive adoption.
---

# Start Project

Start with `orchestrator context audit --json`; absence of a harness means discovery can begin. Keep commands and internal coordination nouns out of normal user-facing replies.

1. Inspect repository facts and existing configuration before asking anything.
2. Ask only one product decision that materially changes the outcome. Offer 2–3 mutually exclusive choices, put the recommendation first, and allow free-form input.
3. Actively surface overlooked security, privacy, accessibility, failure, operations, observability, and data-lifecycle decisions.
4. After each material answer, normalize the complete current product snapshot and run `orchestrator project checkpoint`. Store summaries, roles, journeys, capabilities, policies, scenarios, quality, UI coverage, technology needs, decisions, assumptions, and open questions—never raw conversation or tone.
5. Delay framework and technology selection until requirements justify it; verify current official security, maintenance, and release status when selection becomes necessary.
6. Use `orchestrator project init` for a new root or `orchestrator project adopt` for an existing repository. Review the plan, preserve user files, apply it, then run `context audit` again.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), and [context recovery](../../references/context-recovery.md). If the CLI is unavailable, create or use the repo-local fallback and state the reduced verification coverage.
