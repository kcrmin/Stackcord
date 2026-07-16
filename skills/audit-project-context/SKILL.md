---
name: audit-project-context
description: Use when context was compacted, an agent repeats settled questions, project state feels forgotten, branches or pointers changed, or source and generated summaries may disagree.
---

# Audit Project Context

Rebuild understanding from durable repository truth before any mutation.

1. Read [context recovery](../../references/context-recovery.md) completely and run `orchestrator context audit --json`.
2. Verify repository trust, nearest root, schema compatibility, actual Git/worktree/submodule state, live task provider visibility, branch record, claim, approved specs/contracts, fingerprints, impact graph, and evidence.
3. Do not let last-edited files, task titles, generated summaries, or chat memory override their canonical source. Mark disagreement as stale or unknown.
4. Report current lifecycle gate, approved baselines, active ownership, changed facts, stale dependents, unknown external state, blockers, and one safe next action.
5. Ask only if a genuine product semantic conflict remains after repository evidence is exhausted.

If the CLI is missing, follow the repo-local fallback and explicitly state reduced verification coverage.
