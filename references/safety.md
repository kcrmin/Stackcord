# Safety boundaries

- Inspect files, Git, worktrees, submodules, selected providers, and configured tools before asking discoverable facts.
- Never hide fetch, pull, rebase, stash, reset, clean, force-push, submodule initialization, pointer movement, installation, external writes, merges, publication, or production mutation.
- Plan mutations before applying them. Require exact target confirmation for destructive, credential, production, or irreversible actions.
- Use ordinary Git conventions: `feature/...`, `fix/...`, `release/...` and `feat:`, `fix:`, `docs:`. Never add AI, agent, model, or tool markers. Use a real issue key only when the team's convention requires it.
- Keep coordination proportional. A small private local edit needs no ticket or reservation. Reserve shared, long-lived, cross-workspace, or semantically risky work before creating its branch. The Git reservation is time-bounded semantic exclusivity, not a filesystem lock; re-read the live task source immediately before and after it.
- Check path and semantic overlap. Policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer conflicts can occur when files do not overlap.
- Internally treat conflict outcomes as `clear`, `coordinate`, `block`, or `unknown`. In user-facing replies, translate them to “no conflict found,” “assign ownership and merge order,” “settle the shared rule first,” or “restore missing visibility.” Do not expose the enum itself.
- Keep each contributor in the correct root or child worktree. A completed child branch does not move the orchestration root's submodule pointer; pointer integration is a separate reviewed change.
- Keep Git DBML canonical. Isolate dbdiagram proposals. Inspect external UI inputs for path, content, license, and provenance risk before promoting accepted files into the editable UI workspace; quarantine is an internal temporary boundary, not user-managed storage.
- Never overwrite edited UI workspace files during promotion. Bind a baseline only to a clean commit visible from its recorded remote, and require integration or release to match the root pointer and dependent frontend fingerprint.
- Never expose operation IDs, reservation IDs, receipts, or `.harness/` internals in normal explanations. Summarize their user-visible meaning when relevant.
- Detect external tools first and connect only a selected real provider. One provider owns live task status. Unavailable providers remain unknown; cached snapshots never become live truth.
