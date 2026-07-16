# Workspaces and submodules

Create a workspace as soon as an independent implementation, validation, ownership, or contract boundary is known. Use a submodule when that boundary also needs a separate repository and exact versioned integration; otherwise use root/directory/external.

The root orchestration repository must be cloned because it owns contracts and coordination state. Initialize submodules at root-pinned commits with `git submodule update --init --recursive`; never use `update --remote` as an integration policy.

Before work, inspect root and every workspace for dirty, ahead, behind, diverged, detached, missing, unsafe URL, pointer mismatch, and nested-module state. Use a separate worktree for parallel branches. Claims cover semantic scope; worktrees only isolate files.

For cross-repository changes, merge an additive/versioned contract first, then providers, consumers, frontend connection, and finally the root pointer. Every pointer PR names exact workspace commits, verification, deploy order, and rollback.
