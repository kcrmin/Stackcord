# Context recovery reference

Find the nearest `.harness/manifest.yaml`, establish trust, and run `orchestrator context audit --json`. Read actual filesystem and Git state, the current branch claim, and only referenced specs/contracts. Treat `specs/` as product meaning, `contracts/` as obligations, `.harness/` as coordination state, and task providers as execution status only.

Report facts, stale state, unknown state, blockers, evidence, and one safe next action. Generated summaries never override approved source documents. After compaction, branch changes, pointer changes, or repeated questions, audit again before mutation.
