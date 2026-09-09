# Task management and work reservation

## Keep one live status source

Choose **one live status source** for assignee, workflow status, and visible team progress. The default is Git-local. A team may instead select GitHub Issues, Jira, Beads, or an existing provider only when an authenticated connector or real CLI can read and update it. The project records the choice in `.harness/work/provider.yaml`; it never pretends that an unavailable adapter worked.

Repository work definitions remain the durable executable checklist: outcome, acceptance and failure scenarios, affected workspaces, semantic scope, dependencies, merge order, first failing test, and required evidence. This is not a second task board. The selected provider owns live status; the repository owns product meaning and the implementation boundary.

Use this machinery only when it creates coordination or recovery value. A small private local edit may proceed without a hosted issue or Git reservation. Shared ownership, a long interruption, multiple workspaces, a service rule or contract change, a migration, a shared UI flow, or a likely parallel collision triggers the durable checklist and reservation flow. Before merge, behavior still needs the applicable tests and review regardless of task size.

## Follow the natural-language flow

The contributor says “Build account recovery” or “What should I do next?” The AI then performs this flow without asking the user to manage internal IDs:

1. Inspect the repository, selected provider, Git remote, branches, worktrees, submodules, contracts, and existing decisions.
2. Clarify only product decisions that change the result. For coordinated work, save a normalized executable checklist with `stackcord work define`; for a small private edit, keep the plan in the ordinary change and tests.
3. If an external provider is selected, create or update its visible issue through the chosen connector. Record the issue relationship, assign the intended owner, and move it to the mapped `in_progress` state.
4. Re-read the exact issue. Normalize its item ID, revision, status, owner, dependencies, capabilities, work fingerprint, fetch time, and payload hash, then run `stackcord work provider reconcile --apply`.
5. For coordinated work, run `stackcord work start --apply`. The CLI compares path, policy, scenario, contract, DB entity, migration, UI flow, dependency, and submodule-pointer scope, then uses compare-and-swap on the Git coordination branch. Only after that reservation succeeds does the AI create the conventional branch or worktree.
6. Develop with the checklist and TDD. Before status or ownership changes, verify evidence first, update the external provider through its connector, re-read and reconcile the new revision, then synchronize it with `stackcord work transition --apply` or `stackcord work handoff --apply`.

An issue assignment is not an exclusive lock. GitHub can represent multiple assignees, and issue systems do not understand service contracts or DB meaning. The Git compare-and-swap reservation supplies cross-repository semantic exclusivity without becoming another live status source.

## Choose the provider that fits

| Provider | Good fit | Trade-off |
| --- | --- | --- |
| Git-local | A new project, small team, offline or provider-neutral workflow | No hosted board; Git remote is required for a team-safe reservation |
| GitHub Issues | Teams already reviewing and planning on GitHub | Requires a real authenticated GitHub connector; assignee state alone is advisory |
| Jira | Teams with established workflows, permissions, and reporting | Requires a real Jira connector such as a selected Atlassian integration; field and status mapping must be explicit |
| Beads | Teams deliberately choosing a Git-friendly distributed task graph | Requires the detected Beads CLI and its own operational model; it is optional, not bundled |
| Existing provider | A repository with another working system | Select it only when live reads, writes, ownership, dependency mapping, and revision evidence are observable |

GitHub Issues is not fixed as the default, and Jira or Beads is not enabled merely because its name appears in a document. The AI first detects installed connectors and CLIs, checks current official maintenance and security information when selection matters, compares two or three realistic candidates, explains trade-offs, and connects only the user's choice.

## Keep repository Git conventions

Tell Stackcord the branch, commit, pull-request, and issue conventions once. The `use-git-conventions` Skill first checks `CONTRIBUTING.md`, `AGENTS.md`, and existing pull-request or issue templates. When the rules are consistent, it stores only their normalized form in `.harness/git-conventions.yaml` and reuses it before later Git work.

```yaml
schema_version: 1
branch:
  format: "{type}/{issue}-{description}"
  types: [feat, fix]
commit:
  title_format: "{type}({scope}): {subject}"
  types: [feat, fix]
  scopes: [api, ui]
  max_length: 72
pull_request:
  title_format: "[{issue}] {title}"
  required_sections: [Summary, Test plan]
issue:
  title_format: "[{type}] {title}"
  required_sections: [Problem, Acceptance]
```

The file is optional. Without it, existing Stackcord branch behavior remains unchanged and the AI does not invent commit, pull-request, or issue requirements. Repository conventions never relax safe Git-reference checks or authorize issue and pull-request writes. If contribution files conflict, the AI reports the conflict and asks which rule should win.

## Know what is stored

- `.harness/work/definitions/<work-id>.yaml` is the committed executable checklist and semantic scope.
- `.harness/work/mappings/<work-id>.yaml` is the committed stable relationship to the selected external item.
- `.harness/local/providers/<provider>/<work-id>.yaml` is an ignored, short-lived normalized observation. It is never canonical and a stale or cached copy cannot authorize a mutation.
- The remote coordination branch contains compact time-bounded semantic reservations updated with Git compare-and-swap. Users do not edit it directly.
- The selected external issue contains the human-visible assignee, workflow status, and team discussion.

No raw provider payload, token, raw conversation, or user speaking style is committed.

## Recover after clone or interruption

A clone restores work definitions, mappings, contracts, decisions, workspace topology, and the remote Git reservation. It deliberately does not restore ignored provider observations. The AI can therefore recover who reserved which branch and meaning, while reporting external status as unknown until the selected connector re-reads it. A cached snapshot never becomes live truth simply because it exists.

If a contributor runs out of tokens, changes computers, or returns later, they say “Continue this project.” The AI audits repository evidence, refreshes the provider, reports confirmed, stale, unknown, blocked, and local-only state, then recommends one dependency-ready action.

## Handle failure without inventing state

If the provider is unavailable, the AI reports unknown and offers reconnection or an explicit decision to switch the single provider. It does not silently copy status into Git-local. If the provider update succeeds but Git reservation compare-and-swap loses a race, no branch work begins; the AI refreshes ownership and conflict scope. If the external status changes without matching semantic coordination, integration and release remain blocked until the exact revisions agree.

Branches, commits, and pull requests use the team's stored conventions, for example `feature/account-recovery`, `feat(account): add recovery challenge`, and “Add account recovery flow.” They never include AI, agent, model, or tool branding.

## Live GitHub Issues

`stackcord issues list --repo owner/repository --json` reads live issues. Preview a creation with `stackcord issues create --repo owner/repository --id work-example --title TITLE --body-file issue.md`; add `--apply` to create it. The stable ID prevents duplicate retries, including when the earlier issue was closed. The UI offers the same preview/create flow.

For existing work definitions, supply a JSON mapping such as `{"work.example": 42}` to `stackcord issues migrate --root . --repo owner/repository --mapping mapping.json`, then apply the reviewed migration with `--apply`. Definitions, dependencies and acceptance references remain in the repository; GitHub owns live status. Migration checks the linked issues and dependencies before updating local mappings. Selected GitHub lifecycle paths refresh authenticated issue data; copied snapshots cannot become live truth. Other projects keep their selected provider.
