---
name: start-project-work
description: Use when approved work should begin on a branch or worktree and collaborators need explicit ownership, conflict preflight, and recovery context.
---

# Start Project Work

Begin only from coherent product meaning and observable collaboration state.

1. Run `orchestrator context audit --json`, then `orchestrator work start --json` with the work ID.
2. Verify acceptance refs, contract version, failure behavior, TDD starting test, workspace, baseline, dependencies, and live task status.
3. Check path, policy, scenario, contract, database entity, migration slot, UI flow, dependency, workspace, and root-pointer overlaps. For `unknown`, restore visibility before implementation; for `block`, unify design; for `coordinate`, set explicit ownership and order.
4. Create a conventional `<type>/<description>` branch and isolated worktree when useful. Record a time-bounded claim and branch checkpoint; claims signal intent rather than lock files.
5. Do not embed AI markers in branch or commit names.

Run `context audit` again after any branch, pointer, or baseline change. Read [approval](../../references/approval.md).
