# Context recovery

Find the nearest `.harness/manifest.yaml`, establish repository trust, and run `orchestrator context audit --json`. Then inspect actual Git, worktree, workspace, and exact submodule pointer state without mutation.

Use this source precedence:

1. approved `specs/` product meaning;
2. `contracts/` behavioral and data obligations;
3. actual Git, submodule, filesystem, tests, and artifacts;
4. `.harness/` coordination state;
5. the selected task source for live execution status;
6. generated summaries and chat memory only as navigation hints.

Report facts, stale state, unknown state, blockers, active ownership, evidence, and one safe next action. Audit again after context compaction, repeated settled questions, branch changes, pointer changes, or source/generated disagreement.

If the CLI is unavailable, follow the repository-local Skill and Markdown fallback. State that fingerprint, divergence, remote-claim, semantic-conflict, and release-identity checks have reduced coverage.
