# Tool selection boundary

This product owns durable service meaning, workspace topology, semantic work reservation, deterministic verification, and exact release identity. External tools may improve one part of the workflow but never silently replace those sources.

Before recommending a tool:

1. Inspect installed Skills, Plugins, connectors, CLIs, repository configuration, and the team's existing provider.
2. Decide which missing capability is actually needed: engineering method, formal planning, task status, conversational recall, design collaboration, database visualization, CI, or publication.
3. When the choice can change, search current official documentation and security or release status. Compare two or three viable candidates, including “keep the current tool” when appropriate.
4. Explain why each candidate fits, its operating cost and lock-in, what data leaves the repository, and which source of truth it would own.
5. Ask one material choice as A/B/C with the recommended option first and marked, plus free-form input. Connect or install only the chosen tool with explicit authority, then record the dated decision and reevaluation trigger.

Common boundaries:

| Tool family | What it can add | What this product still owns |
| --- | --- | --- |
| Superpowers | Brainstorming, plans, TDD, debugging, worktrees, review discipline | Product and contract memory, multi-repository truth, semantic reservation, provider reconciliation, release identity |
| BMAD | Formal roles, planning artifacts, structured delivery methods | Actual Git/submodule state, canonical service rules, conflict proof, provider and candidate identity |
| GitHub Issues or Jira | Human-visible assignee, workflow, discussion, reporting | Executable repository checklist and Git CAS semantic reservation |
| Beads | Optional distributed task graph for teams choosing its CLI and storage model | Service contracts, workspace topology, evidence, and release gates |
| Memory or conversation-continuity tools | Faster recall and navigation hints | Canonical repository evidence, fingerprints, stale detection, Git truth, and safety decisions |
| dbdiagram CLI | Isolated DBML visualization and remote diagram collaboration | Git DBML, accepted semantic changes, migrations, contracts, and rollback evidence |
| UI design tools | Mockups and design collaboration | Imported authority (`reference`, `seed`, or `canonical`), provenance, accessibility, and implementation mapping |

Do not recommend a bundle merely because it is popular. Prefer an existing working tool when it meets the need. Do not claim support for Jira, GitHub, Beads, or any future provider unless a real connector can read and write it and produce a fresh normalized observation. Re-run this comparison when the current tool becomes unavailable, materially changes its license or maintenance status, or a new requirement appears.
