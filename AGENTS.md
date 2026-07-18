# Agent entry point

Read `docs/design/index.md` before changing the product. Follow the linked service
or UI specification only when the change affects that area; do not load unrelated
design records into every task.

Implementation rules:

- Use test-first development for behavior changes and bug fixes.
- Keep generated projects framework, language, database, cloud, and Git-host neutral.
- Do not add AI markers to branch, commit, or pull-request names.
- Do not perform hidden pull, rebase, stash, reset, force-push, external write, installation, or release.
- Preserve the distinction between `specs/`, `contracts/`, `.harness/`, and `docs/`.
- English machine identifiers are canonical; keep English and Korean user documentation in semantic parity.
- Public naming and final release require explicit user approval.
