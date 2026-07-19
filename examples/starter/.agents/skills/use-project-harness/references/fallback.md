# Plugin-less and CLI-less fallback

1. Treat the natural-language request as the entry point. Read `AGENTS.md`, `.harness/entry.md`, the manifest, workspaces, profile, and selected task source; do not ask the user to operate internal files.
2. From a child repository, locate the actual orchestration root. Inspect branch, dirty state, upstream, ahead/behind/diverged state, worktrees, workspace commits, remotes, and exact submodule pointers without mutation.
3. Read only related approved `specs/`; product, business, behavior, interface, and data `contracts/`; current work definitions; and test evidence.
4. If an external task source is selected, refresh it with a real authenticated connector or CLI. Treat cached status as unknown. Recover a Git work reservation from the coordination branch, but do not present it as fresh external status.
5. Separate confirmed facts, stale derivations, unknown external state, blockers, active ownership, and local-only work. State one safe next action. Run a context audit when settled questions repeat or sources disagree.
6. A small private local edit needs no ticket or reservation. Before shared or risky work, define the service meaning, behavioral boundary, first failing test, semantic scope, owner, dependencies, and merge order; then synchronize the selected task source and Git work reservation.
7. Require test and integration evidence before merge. Bind technical and user validation to one release candidate. Keep strict release optional.
8. Before protected product meaning becomes canonical, verify the exact commit and fingerprint through `orchestrator governance check --json`. A contributor may prepare a proposal and PR, but Git user.name or user.email never proves product authority. If review state is unavailable, do not integrate or release.

Without the CLI, fingerprint, divergence, atomic remote reservation, semantic-conflict, archive-safety, and exact release-identity verification has reduced coverage. Do not report those checks as passed.
