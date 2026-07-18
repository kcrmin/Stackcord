---
name: continue-project
description: Use when a cloned, transferred, or paused project must be resumed, the next safe work is unclear, or shared and local progress need reconstruction.
---

# Continue Project

Recover from repository evidence, not chat memory. Present one plain-language next action while keeping coordination mechanics in the background.

1. Resolve the CLI from `ORCHESTRATOR_CLI`, a repository build, or `PATH`, then run `orchestrator status --json` before mutation. Resolve the orchestration root when invoked inside a frontend, backend, or other child workspace. On an explicit continuation request, an absent CLI may trigger an offer to run the matching verified `scripts/bootstrap-cli.sh` or `scripts/bootstrap-cli.ps1` with an explicit release URL, version, and install directory; lifecycle hooks never install it.
2. Re-read actual branch, dirty state, upstream, ahead/behind/diverged state, worktrees, workspace commits, root submodule pointers, approved product meaning, service contracts, evidence, and selected live task provider.
3. If GitHub, Jira, or Beads is selected, use only its installed authenticated connector or CLI. Reconcile with `orchestrator work provider`; when unavailable, mark live state unknown and offer reconnection or reduced Git-local work. Never present a cached snapshot as live.
4. Summarize confirmed facts, stale derivations, unknown external state, blockers, active ownership, and local-only work. Do not expose IDs or internal storage paths unless diagnosing them is necessary.
5. Never auto-pull, rebase, stash, reset, initialize submodules, move pointers, or claim that unpushed commits are shared.
6. When state is coherent, run `orchestrator work next --json`. Exclude work blocked by unresolved policy, scenario, contract, migration, UI, dependency, ownership, evidence, or pointer conflicts.
7. Recommend one small reviewable next action using user value, dependency order, unblock impact, and risk. Name the affected workspace, acceptance reference, and first failing test. Ask only when equal safe choices change product intent or Git reconciliation needs human direction.

Example: after “이 프로젝트 이어서 해,” report whether frontend, backend, root pointers, and task status are current; then recommend one dependency-ready change instead of restarting discovery.

Read [context recovery](../../references/context-recovery.md), [workflow](../../references/workflow.md), and [safety](../../references/safety.md). If the CLI is absent, follow the repo-local fallback and state which checks remain unverified.
