---
name: handoff-project-work
description: Use when responsibility for active work actually changes, a contributor must pause unexpectedly, or another person needs an explicit recovery checkpoint.
---

# Handoff Project Work

Use handoff only for ownership transfer; ordinary collaboration uses claims and shared canonical context.

1. Run `orchestrator context audit --json` and `orchestrator work handoff --json`.
2. Capture work/claim IDs, branch/worktree, exact commits and pointers, dirty or uncommitted state, completed evidence, failed tests, blockers, decisions, unknowns, and one reproducible next action.
3. Link stable specs/contracts and paths rather than copying product policy into the handoff. Never copy secrets or raw conversation.
4. Ask one question if the receiving owner or treatment of local-only changes is unclear. Never silently stash, commit, push, discard, or transfer credentials.
5. Expire or transfer the old claim only after the receiving scope and observable checkpoint are confirmed.

Read [context recovery](../../references/context-recovery.md) and [approval](../../references/approval.md).
