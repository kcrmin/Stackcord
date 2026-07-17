# Core concepts

## Conversation and verification

Skills interpret natural-language goals, discover product meaning, choose what to ask, and explain trade-offs. The CLI does not replace that judgment; it verifies observable state and reproducible identity. It is read-only or plan-first unless an explicit, visible apply step is requested.

## Sources of truth

Actual Git, submodule, worktree, and filesystem state outrank cached summaries. `specs/` owns product intent and policy. `contracts/` owns cross-component behavior and failures. Git-tracked DBML owns data structure. `.harness/` indexes those sources and records compact coordination state. One selected task provider owns live task status; Git-local is the default, not a mandate.

## Identity and staleness

Product facts receive stable IDs so references survive file movement and rewriting. Fingerprints bind indexes, change plans, evidence, and release candidates to exact content. When a dependency fingerprint changes, downstream state is stale until refreshed or deliberately accepted; timestamps alone are not trusted.

## Workspaces and children

A workspace is a repository or separately testable component represented in the root harness. A child workspace is commonly a Git submodule with its own history, branch, tests, and release responsibility. The root records the exact accepted child commit. A frontend and backend do not need to be separate merely because of their names; split them when independent ownership or lifecycle justifies it.

## Conflict model

Filesystem overlap is only one conflict type. Claims also reserve policies, scenarios, contracts, DB entities, migration slots, UI flows, dependency majors, stable IDs, and root pointers. A detected conflict blocks silent parallel editing and creates a discussion about ownership, boundaries, or sequence; it does not perform an automatic reset, stash, or rebase.

## Iterative delivery and TDD

The team establishes whole-product meaning and UI coverage, then delivers role/domain/journey slices continuously. Shared interfaces, contracts, and schemas are agreed early when they reduce parallel ambiguity. Behavior, bugs, contract changes, migrations, UI interactions, and integration rules start with a failing test or reproducible failing check and end with passing evidence.

## Context recovery

After clone, session restart, or context compression, the AI reads `AGENTS.md` and `.harness/entry.md`, audits canonical files, checks actual Git and workspace state, and recommends one safe next action. It must not ask questions already answered by valid current sources. If the Plugin is absent, `.agents/skills/use-project-harness/` provides the same recovery entry with reduced convenience.

## Release identity

Core release preparation binds source commits, artifact digests, product/docs/contract fingerprints, TDD evidence, integration evidence, and applicable migration evidence into one candidate digest. User validation is collected after the candidate exists and must name that exact digest. Strict release adds supply-chain evidence and protected publication tooling as an opt-in profile.
