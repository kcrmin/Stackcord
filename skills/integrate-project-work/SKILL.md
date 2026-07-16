---
name: integrate-project-work
description: Use when branches, pull requests, workspace repositories, submodule pointers, contracts, or frontend/backend implementations are ready to combine.
---

# Integrate Project Work

Integrate a verified compatibility bundle instead of pretending repositories merge atomically.

1. Run `orchestrator context audit --json` and `orchestrator integrate plan --json`.
2. Verify branch ownership, clean state, current remote tracking, TDD evidence, required reviews/checks, contract compatibility, migrations, generated artifacts, and deployment order.
3. Merge shared additive contracts first. Then integrate providers, consumers, and UI connection in the approved order. Multiple backend implementations may merge and pass contract tests before frontend connection when that reduces conflict.
4. Update the root submodule pointer only to reviewed exact workspace commits. Never follow a remote branch implicitly or move a dirty/detached workspace.
5. Re-run cross-workspace and root verification after pointer integration; preserve rollback commits and receipts.

Use [approval](../../references/approval.md) for push, PR, merge, or external writes. Run `context audit` after each integration boundary.
