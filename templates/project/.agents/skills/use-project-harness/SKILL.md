---
name: use-project-harness
description: Use when starting, continuing, changing, coordinating, recovering, or releasing work in this repository.
---

# Use Project Harness

Run `orchestrator status --json`, then read only the canonical sources related to the request. If invoked in a child repository, resolve the actual orchestration root before claiming service-wide context. Treat `specs/` as approved product meaning and `contracts/` as service purpose, guarantees, non-goals, business rules, failure behavior, APIs, events, and data obligations.

Ask one material product question at a time; infer facts from files, Git, and selected tools. Checkpoint normalized meaning after each material answer, never raw dialogue. Use TDD for behavior, bugs, contracts, migrations, and UI interactions. Before parallel work, re-read the selected task source, check path and semantic scope, claim shared work, set ownership and merge order, and use conventional Git names without AI markers.

Keep coordination internals out of normal replies. If context was compacted, settled questions repeat, or sources disagree, run `orchestrator context audit --json` before mutation. If the CLI is unavailable, follow `references/fallback.md` and state reduced verification.
