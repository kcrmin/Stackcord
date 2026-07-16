---
name: manage-contract-change
description: Use when API, event, data, authentication, error, retry, idempotency, compensation, or service obligations may change across components.
---

# Manage Contract Change

Treat contracts as behavioral obligations, not only payload schemas.

1. Run `orchestrator context audit --json`, `orchestrator contract impact --json`, and compatibility checks.
2. Identify providers, consumers, policy/scenario refs, generated clients/mocks, tests, deployments, and rollback paths.
3. Classify additive optional changes as compatible. Treat removal, required additions, type narrowing, error semantic, retry, idempotency, and compensation changes as breaking.
4. Prefer side-by-side versions: merge the additive contract, update providers, verify consumers, migrate consumers, then retire the old version. Ask one question only when product obligation is undecided.
5. Record exact merge/deploy order and compatibility evidence before implementation or integration.

Use `contract impact` after each revision and [context recovery](../../references/context-recovery.md) after compaction.
