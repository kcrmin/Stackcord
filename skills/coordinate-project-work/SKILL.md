---
name: coordinate-project-work
description: Use when contracts, DBML, dbdiagram, external UI, ownership transfer, branches, repositories, submodules, or integration order must be coordinated.
---

# Coordinate Project Work

Start with `orchestrator context audit --json`. Select only the relevant path below and keep coordination internals out of normal user explanations.

- Contract or failure behavior: run `orchestrator contract impact --json` and `contract check`. Treat errors, retry, idempotency, compensation, auth, and service obligations as contract meaning. Prefer additive or side-by-side compatibility.
- Database: keep Git DBML canonical; run `orchestrator db diagram --json` and `db diff`. Quarantine dbdiagram edits, show the semantic entity/column/relation/index/note difference, and ask why meaning changed before canonical writes. Include migration and rollback evidence.
- External UI: run `orchestrator ui import --json` in plan mode. Reject unsafe archives, secrets, unclear licensing, and missing provenance. If authority is unstated, ask whether it is `reference`, `seed`, or `canonical`, recommending the least authority that fits.
- Active ownership transfer: use `orchestrator work handoff --json` only when responsibility changes. Capture exact commits, pointers, local-only state, evidence, blockers, and one reproducible next action; never silently stash, commit, push, or discard.
- Integration: run `orchestrator integrate plan --json`. Verify clean exact states, TDD evidence, compatibility, migrations, generated artifacts, and rollback. Integrate additive contracts, providers, consumers, UI connection, then the reviewed exact root submodule pointer.

For overlapping work, explicitly assign semantic/path ownership and merge order before implementation. Multiple backend implementations may integrate against contract tests before frontend connection when that lowers conflict.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), and [context recovery](../../references/context-recovery.md). Run `context audit` after each canonical integration boundary.
