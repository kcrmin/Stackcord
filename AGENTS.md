# Agent entry point

Read these files before changing the product:

1. `docs/design/index.md`
2. `docs/design/11-cross-review-and-confirmation.md`
3. `docs/superpowers/plans/2026-07-16-fullstack-orchestrator-production.md`

Implementation rules:

- Use test-first development for behavior changes and bug fixes.
- Keep generated projects framework, language, database, cloud, and Git-host neutral.
- Do not add AI markers to branch, commit, or pull-request names.
- Do not perform hidden pull, rebase, stash, reset, force-push, external write, installation, or release.
- Preserve the distinction between `specs/`, `contracts/`, `.harness/`, and `docs/`.
- English machine identifiers are canonical; keep English and Korean user documentation in semantic parity.
- Public naming and final release require explicit user approval.
