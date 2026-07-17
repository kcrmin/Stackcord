# Troubleshooting

## The AI forgot the project

Say “Recover this project context before doing anything.” The recovery Skill reads `AGENTS.md`, `.harness/entry.md`, canonical specs and contracts, then runs context and Git audits. If it repeats answered questions, ask for a context audit and the exact source or fingerprint behind each unknown. Do not reconstruct the project from conversation memory.

## Clone has missing or mismatched submodules

Ask for a Git inspection and submodule sync plan. Missing checkout is different from a pointer mismatch, dirty child, detached child, or unreachable commit. Initialize only the root-recorded commit. Commit and publish legitimate child changes before changing the root pointer.

## A branch is dirty or diverged

Stop automatic mutation. Ask the AI to show branch, upstream, ahead/behind counts, changed paths, and commits unique to each side. Choose merge, rebase, commit, stash, or cleanup only after seeing the impact. The product never silently resets or force-pushes.

## Parallel work is blocked by a conflict

Read the conflict category. For path overlap, split files or serialize. For policy, contract, DB, UI, dependency, or pointer overlap, agree on the shared boundary and integration order. Update the claims only after ownership is clear; do not delete them merely to bypass a blocker.

## An external task tool is unavailable

Keep Git-local as live status until a real connector or executable is available. Do not copy status between multiple authorities. Product specs, contracts, claims, fingerprints, and release identity remain repository-owned regardless of the task provider.

## DBML or UI input is stale or unsafe

Keep external input in quarantine. Compare semantics and provenance, confirm license and rationale, then explicitly promote accepted changes. Never let a visualization, archive, or remote mockup overwrite canonical files automatically.

## Release verification no longer passes

Compare the reported changed field with current commits, artifacts, product docs, contracts, tests, integration results, migrations, and user validation. Any material change creates a new candidate and requires validation of its new digest. Do not edit a digest or validation record to make it pass.

## Plugin is unavailable

Open `.agents/skills/use-project-harness/SKILL.md` and its Markdown fallback with any capable coding AI. Build or locate the Go CLI when deterministic checks are required. Plugin-less operation has reduced convenience, not a different source of truth.
