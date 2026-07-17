# DBML and dbdiagram

## Keep one canonical model

Store canonical DBML in Git with stable entity IDs and review it like code. Contracts and product policies describe behavior; DBML describes physical data structure; migrations describe ordered transitions. A rendered diagram is not a second source of truth.

## Discuss the model with the AI

Ask the AI to propose or revise the data model from policies, scenarios, retention, privacy, access, failure, and scale requirements. It writes DBML and runs semantic checks. Review names, ownership, cardinality, nullability, uniqueness, lifecycle, deletion, auditability, and sensitive-data boundaries before implementation.

## Visualize in isolation

When the user selects the [official dbdiagram CLI](https://docs.dbdiagram.io/release-notes/2026-07/), the AI detects the `dbdiagram` executable and creates an operation-scoped local workspace under `.harness/local/dbdiagram/`. It copies canonical DBML to `candidate.dbml`, prepares `dbdiagram init --entry candidate.dbml --diagram-id <id>`, and then explicitly runs `dbdiagram push` or `dbdiagram pull` only after the external action is visible. Credentials remain outside Git. Visualization or remote collaboration never modifies canonical DBML implicitly.

## Reconcile remote changes

`push` updates the selected online diagram from the isolated copy so collaborators can view it. If someone changes that diagram externally, `pull` updates only the isolated copy. Compare entity/field/index/relation semantics and ask why material changes were made. Accepted differences become an explicit Git change with updated policy, contract, and migration impact. Rejected differences leave canonical DBML unchanged.

## Evolve production data safely

Every destructive or compatibility-sensitive difference requires a migration sequence, consumer compatibility plan, validation, backup or rollback strategy, and TDD evidence. Reserve migration slots before parallel work. Release evidence is required only when a migration is actually part of the candidate.
