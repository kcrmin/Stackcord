# Concepts

The harness is the whole durable collaboration structure, not only `.harness/`: `specs/` owns product meaning, `contracts/` owns cross-component obligations, `.harness/` owns coordination and evidence, and `docs/` owns guides and runbooks.

A workspace is an implementation, validation, ownership, and contract boundary. Its kind can be root, directory, submodule, or external. A child is simply a nested agent or process; it is not a project structure concept. Submodules are recommended when a newly identified boundary genuinely needs a separate repository and exact pinned commit.

Stable IDs such as `policy.account.recovery.rate-limit` survive file moves. A ticket number and a branch description are execution identifiers, not product meaning. Claims declare who intends to change paths, policies, contracts, migrations, UI flows, dependencies, and pointers; they are not distributed locks.

Lifecycle stages are dependency gates, not waterfall deadlines. Product-wide intent and UI coverage are established first, but integrated in small role/domain/journey changes. Shared interfaces and failure semantics precede parallel implementations; TDD then drives vertical slices.

The Plugin packages Skills and optional Hooks. The repo-local Skill preserves continuation without the Plugin. The CLI performs schema, Git, operation, conflict, adapter, and release checks. Hooks only remind trusted sessions to audit context; they do not write or call external systems.
