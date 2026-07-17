---
name: use-project-harness
description: Use when starting, continuing, changing, coordinating, recovering, or releasing work in this repository.
---

# Use Project Harness

Read `.harness/entry.md`, inspect actual Git/workspace/submodule state, and run `orchestrator context audit --json` when available. Treat `specs/` as product meaning and `contracts/` as behavioral obligations. Read only the sources related to the current request.

Ask one material product question at a time; infer facts from files and Git. Use TDD for behavior, bugs, contracts, migrations, and UI interactions. Before parallel work, check path and semantic scope, set ownership and merge order, and use conventional Git names without AI markers.

Keep coordination internals out of normal replies. If context was compacted, settled questions repeat, or sources disagree, audit again before mutation. If the CLI is unavailable, follow `references/fallback.md` and state reduced verification.
