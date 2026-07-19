# Focused product design

The product has one governing boundary: **flexible AI Skills own conversation and product judgment; the deterministic CLI owns actual-state, safety, conflict, and identity checks.**

## Kept in the core

- Continuous normalized discovery checkpoints and framework-neutral init/adopt.
- Repo-local Skill and Markdown fallback for clone and context recovery.
- Stable IDs, fingerprints, stale propagation, and real Git/submodule/worktree diagnosis.
- Path and semantic conflict claims, contract/DBML/UI coordination, TDD, integration order, and exact-candidate release verification.
- Go builds for macOS and Windows and paired English/Korean user documentation.

## Isolated in strict release

SBOM, provenance, signatures, supply-chain evidence, protected publication checks, and organization-oriented evidence are under `profiles/strict-release/`. They are security-relevant extensions, not everyday lifecycle requirements.

## Removed or deferred

Unused provider registries, claims of unsupported Jira/Linear/Beads adapters, duplicate Skills, repeated policy prose, user-managed operation identifiers, default publication commands, mandatory package-manager distribution, and future-only abstraction layers were removed from the core.

## Authoritative design records

- [Service continuity harness specification](../superpowers/specs/2026-07-18-service-continuity-harness-design.md)
- [Editable UI workspace specification](../superpowers/specs/2026-07-18-ui-baseline-submodule-design.md)
- [Stackcord product naming](../superpowers/specs/2026-07-19-stackcord-naming-design.md)
- [Stackcord README information design](../superpowers/specs/2026-07-19-readme-information-design.md)
- [Core concepts](../concepts/en.md)
- [Korean core concepts](../concepts/ko.md)

Completed implementation history lives in Git. Historical plans are not active
product instructions.
