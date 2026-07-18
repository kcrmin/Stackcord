# Plugin-less and CLI-less fallback

1. Read `AGENTS.md`, `.harness/entry.md`, `.harness/manifest.yaml`, `.harness/workspaces.yaml`, and `.harness/profile.yaml`. From a child repository, inspect `.harness/bridge.yaml` and locate the orchestration root before claiming full service context.
2. Inspect the current root, branch, dirty state, upstream, worktrees, workspace commits, and exact submodule pointers without mutation.
3. Read related approved `specs/`; service, failure, API, event, and data `contracts/`; the selected task source; active ownership; and available test evidence.
4. Separate confirmed facts, stale derivations, unknown external state, blockers, active ownership, and local-only work. State one safe next action.
5. Before changing work, define affected product meaning, behavioral interface, failing test, path/semantic ownership, live claim, and integration order.

Without the CLI, fingerprint, divergence, remote-claim, semantic-conflict, archive-safety, and release-identity verification has reduced coverage. Do not claim those checks passed. Never treat a cached external-provider snapshot as current.
