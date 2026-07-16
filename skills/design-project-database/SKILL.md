---
name: design-project-database
description: Use when data structure, DBML, migrations, retention, ownership, or dbdiagram visualization and synchronization need product-aware design.
---

# Design Project Database

Keep Git DBML canonical and use dbdiagram as a reviewed visual collaboration adapter.

1. Run `orchestrator context audit --json` and `orchestrator db diagram --json`.
2. Derive entities, relationships, constraints, lifecycle, tenancy, privacy, retention, deletion, audit, concurrency, and failure behavior from approved journeys and policies. Ask one material question at a time with a recommended choice.
3. Write DBML in `contracts/data/`, validate it, and show the diagram through the configured adapter. Never store credentials in repository files.
4. Pull visual edits only to `.harness/local/dbdiagram/<operation-id>/`. Compute a semantic table/column/relation/index/note diff, identify migration/contract/policy effects, and ask why remote meaning changed before proposing a canonical write.
5. Use expand/migrate/contract migrations with rollback evidence.

Run `context audit` after approved DBML changes. Read [approval](../../references/approval.md) before external push or pull.
