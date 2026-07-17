---
name: recover-and-release-project
description: Use when context was compacted or forgotten, settled questions repeat, canonical sources disagree, production readiness is requested, or a release candidate must be verified.
---

# Recover and Release Project

Start with `orchestrator context audit --json`. After compaction or suspected forgetting, do not mutate anything until facts, stale state, unknown state, active ownership, and one safe next action have been reconstructed.

1. Follow [context recovery](../../references/context-recovery.md). Never let chat memory, task titles, last-edited files, or generated summaries override approved sources and actual Git state.
2. For production readiness, verify product/UI/contract coverage, observable failure behavior, clean exact root/workspace/submodule commits, TDD and integration evidence, security, accessibility, observability, operations, backup/restore, support, and migration/rollback only when applicable.
3. Run `orchestrator release prepare --json` to freeze one candidate digest from the exact inputs. Any included identity change creates a new candidate.
4. Bind technical confirmation and user validation to that same digest, then run `orchestrator release verify --json`. Validation from another build or commit does not count.
5. Use the default `core` profile for ordinary projects. Enable `strict-release` only when the team explicitly needs SBOM, provenance, signatures, organization approvals, and hardened publication checks. Strict adds gates; it never replaces core checks.
6. Publication remains an explicit external action after verification; do not infer it from “finish” or “release”.

Read [workflow](../../references/workflow.md) and [safety](../../references/safety.md). If the CLI or Plugin is missing, use the repo-local fallback and clearly state reduced verification coverage.
