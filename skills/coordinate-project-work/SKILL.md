---
name: coordinate-project-work
description: Use when service rules, contracts, DBML, dbdiagram, external UI, ownership, repositories, submodules, or integration order cross implementation boundaries.
---

# Coordinate Project Work

Coordinate shared meaning before parallel implementation. Run `orchestrator status --json`, re-read the selected task provider, and choose only the relevant path below.

- **Service and technical contracts:** Treat purpose, guarantees, non-goals, business rules, authorization, failure, retry, idempotency, compensation, support, API behavior, events, and data obligations as contracts. Keep human-readable policy meaning separate from machine-readable interfaces, but bind both to stable IDs. Run `orchestrator contract impact --json` and `orchestrator contract check`; prefer additive or side-by-side compatibility.
- **Semantic conflict:** Run `orchestrator work conflict --json` even when files differ. Check policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer meaning. Resolve shared contracts first; then assign path and semantic ownership plus merge order.
- **Database:** Keep Git DBML canonical. Use `orchestrator db diagram --json` to create an isolated dbdiagram proposal and `orchestrator db diff` for semantic entity, column, relation, index, and note changes. Treat direct dbdiagram edits as proposals, explain the difference, and ask why meaning changed before canonical apply. Require contract, migration, rollback, and test references for canonical changes.
- **External UI:** Use `orchestrator ui import --json` to quarantine and inspect archives, provenance, license, secrets, executables, paths, and accessibility risks. Set authority to `reference`, `seed`, or `canonical`, recommending the least authority that fits. Reconcile mappings, then use `orchestrator ui integrate` only for reviewed scoped changes.
- **External lifecycle and ownership:** Verify local evidence before asking the selected connector to change status or owner. After the external write, re-read the exact item and run `orchestrator work provider reconcile --apply`. Then use `orchestrator work transition --apply` or `orchestrator work handoff --apply` to synchronize the Git semantic reservation. A provider assignment is advisory until the Git compare-and-swap reservation agrees. Do not silently stash, commit, push, or discard.
- **Integration:** Use `orchestrator integrate plan --json`, review the exact order, apply, then run `orchestrator integrate verify`. The plan binds both the selected-provider revision and Git semantic-reservation revision. Integrate compatible contracts, provider workspaces, consumer workspaces, UI, migrations, and finally the reviewed root submodule pointer. Never move a pointer merely because a child branch finished.

Require TDD and integration evidence for behavior, bugs, contracts, migrations, and UI interactions. Explain user impact and required coordination in ordinary language. Say “작업 선점” or “work reservation,” never `claim`. Translate internal `clear`, `coordinate`, `block`, and `unknown` outcomes into the concrete user action; keep internal IDs and storage paths hidden unless troubleshooting needs them.

Read [workflow](../../references/workflow.md), [safety](../../references/safety.md), [context recovery](../../references/context-recovery.md), and [tool selection](../../references/tool-selection.md). Rerun combined status after each canonical integration boundary.
