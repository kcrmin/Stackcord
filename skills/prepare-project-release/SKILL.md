---
name: prepare-project-release
description: Use when a project should enter production hardening, create a release candidate, undergo user validation, or publish a production release.
---

# Prepare Project Release

Freeze and verify one exact candidate; any material change creates a new candidate.

1. Run `orchestrator context audit --json` and `orchestrator release prepare --json`.
2. Require approved product/UI/contract baselines, clean exact root/workspace/submodule commits, tests, security checks, licenses, SBOM, signatures/provenance, observability, operations, backup/restore, migration/rollback, and support readiness.
3. Create an immutable RC manifest with artifact checksums. Run CI and strict local validation against that same identity.
4. Ask the user to validate the same RC artifact in their real environment and record exact confirmation. Never translate “looks fine” from a different build into approval.
5. Run `release publish` only with exact production target confirmation. If anything changes, invalidate approval and prepare a new RC.

Read [approval](../../references/approval.md) and [lifecycle](../../references/lifecycle.md).
