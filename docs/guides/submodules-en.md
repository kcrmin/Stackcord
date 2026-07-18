# Submodules, worktrees, and collaboration

## Decide workspace boundaries

Use a child repository when ownership, permissions, dependencies, deployment, or release lifecycle is meaningfully independent. Do not split a simple project solely to imitate a frontend/backend shape. Register each child in `.harness/workspaces.yaml` and Git `.gitmodules`; the root repository remains the coordination and contract boundary.

## Clone and diagnose

After clone, ask the AI to continue the project. It compares `.gitmodules`, the root index pointer, initialized module paths, child HEADs, dirtiness, detached state, and remote reachability. A safe sync plan uses the exact root-recorded commit. Missing, dirty, mismatched, or unpublished child state is explained instead of silently replaced.

## Isolate parallel work

One contributor normally owns one conventional branch per change. Use Git worktrees when the same clone must hold simultaneous branches. For coordinated work, reserve the affected paths, policies, scenarios, contracts, DB entities, migrations, UI flows, dependency majors, and pointer intent. The Git reservation provides semantic exclusivity; the selected task provider remains the only live team-status source. Neither replaces a conversation about shared meaning.

## Integrate in dependency order

When a shared boundary changes, agree on the additive contract first. Merge providers before consumers when possible, then connect the frontend or other consumers after compatibility is available. Commit and review child work in the child repository. Update and review the root pointer only when the chosen child commit is reachable and ready for coordinated integration.

## Handle conflict scenarios

- Same file, different meaning: split the file or serialize the edits.
- Same contract or policy: one owner evolves the boundary; other work waits or targets a compatible version.
- Same DB entity or migration slot: agree on migration order and rollback before implementation.
- Same UI flow: agree on state ownership and acceptance behavior.
- Dependency major overlap: merge one upgrade baseline before feature work.
- Root pointer overlap: designate one integration owner after child merges.
- Dirty/diverged branches: stop, show exact commits and changes, and let the user choose pull, rebase, merge, or cleanup.

## Use handoff deliberately

A handoff is for a real change of ownership, interruption, or unavailable contributor. It records current intent, evidence, blockers, and exact repository identity so the next owner does not reconstruct work from chat. Normal parallel contributors keep their own scopes and share common context through canonical specs, contracts, and Git.
