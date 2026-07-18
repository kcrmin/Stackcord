---
name: recover-and-release-project
description: Use when context was compacted or forgotten, settled questions repeat, canonical sources disagree, production readiness is requested, or an exact release candidate needs validation.
---

# Recover and Release Project

Recover from canonical evidence before mutation, and release only the exact candidate that both technical checks and the user validated.

1. Run `orchestrator status --json`. If questions repeat, context was compacted, or sources disagree, stop mutation and run `orchestrator context audit --json`.
2. Reconstruct approved product meaning, service contracts, actual root and child Git state, exact submodule pointers, selected provider state, active ownership, evidence, and candidate identity. Report confirmed facts, stale derivations, unknown external state, local-only work, blockers, and one safe next action. Chat memory and generated summaries are navigation hints only.
3. For production readiness, verify role/journey/UI coverage, service guarantees and failure behavior, clean published root and workspace commits, exact pointers, TDD and integration evidence, security, privacy, accessibility, observability, operations, support, backup/restore, and applicable migration/rollback evidence.
4. Run `orchestrator release prepare --json` with the selected work and release version. The candidate binds commits, remotes, pointers, provider revisions, contract fingerprints, evidence, migrations, tool versions, and profile; any identity change creates a different candidate.
5. Store user validation evidence outside tracked product files or under ignored local state. Run `orchestrator release validate --evidence <file> --confirm --apply`, then `orchestrator release verify --json`. Block validation from a different digest, build, commit, or stale screen.
6. Use the default `core` profile for ordinary services. Enable `strict-release` only when the organization explicitly needs stronger SBOM, provenance, signatures, approval records, and publication controls. Strict adds gates without replacing core checks.
7. Treat publication, marketplace submission, package-manager release, signing, and production mutation as explicit external actions after verification. Never infer them from “완료” or “release 해줘.”

Read [context recovery](../../references/context-recovery.md), [workflow](../../references/workflow.md), and [safety](../../references/safety.md). If the Plugin is absent, use the repo-local Skill; if the CLI is absent, follow its Markdown fallback and name the checks that remain unverified.
