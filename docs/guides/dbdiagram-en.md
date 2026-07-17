# DBML and dbdiagram

## Keep one canonical model

Store canonical DBML in Git with stable entity IDs and review it like code. Contracts and product policies describe behavior; DBML describes physical data structure; migrations describe ordered transitions. A rendered diagram is not a second source of truth.

## Discuss the model with the AI

Ask the AI to propose or revise the data model from policies, scenarios, retention, privacy, access, failure, and scale requirements. It writes DBML and runs semantic checks. Review names, ownership, cardinality, nullability, uniqueness, lifecycle, deletion, auditability, and sensitive-data boundaries before implementation.

## Visualize in isolation

When dbdiagram CLI or another supported renderer is available, the AI creates an operation-scoped local workspace under `.harness/local/dbdiagram/`. Credentials remain outside Git. Visualization or remote collaboration never modifies canonical DBML implicitly.

## Reconcile remote changes

If someone changes a diagram externally, import it into the isolated workspace, compare entity/field/index/relation semantics, and ask why material changes were made. Accepted differences become an explicit Git change with updated policy, contract, and migration impact. Rejected differences leave canonical DBML unchanged.

## Evolve production data safely

Every destructive or compatibility-sensitive difference requires a migration sequence, consumer compatibility plan, validation, backup or rollback strategy, and TDD evidence. Reserve migration slots before parallel work. Release evidence is required only when a migration is actually part of the candidate.
