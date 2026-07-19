---
name: use-project-harness
description: Use when starting, continuing, changing, coordinating, recovering, or releasing work in this repository.
---

# Use Project Harness

Treat the user's natural-language request as the entry point; do not make them memorize commands or edit `.harness/`. Read `.harness/entry.md`, run `stackcord status --json` when available, and inspect actual Git, workspace, and submodule state. From a child repository, resolve the actual orchestration root before asserting service-wide context. Read only canonical sources related to the request. `specs/` owns product meaning; `contracts/` owns service purpose, commitments, non-goals, business rules, failure behavior, interfaces, and data obligations.

Ask one material product question at a time, infer discoverable facts, and checkpoint normalized decisions rather than raw dialogue. Keep work management proportional: a small private local edit does not need a ticket or Git work reservation. For shared, long-lived, cross-workspace, or semantically risky work, the selected task source owns live status and the Git work reservation owns exclusive semantic scope. Re-read both, check path and meaning overlap, and set ownership and merge order before parallel work. Use conventional Git names without AI markers.

Use TDD for behavior, bugs, contracts, migrations, and UI interactions; exploratory spikes may stay unmerged until evidence exists. Keep coordination internals out of normal replies. If context was compacted, settled questions repeat, or sources disagree, run a context audit before mutation. Use core release normally and enable strict release only for an explicit organizational need. If the CLI is unavailable, follow `references/fallback.md` and state reduced verification.

Before changing service purpose, policy, business rules, contracts, or governance, run `stackcord governance check --json`. If governance is enabled and the selected Git provider does not identify the current account as a product authority, keep the protected change as a proposal and request review. Git user.name and user.email are display metadata, not authority.
