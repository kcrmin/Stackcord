# Product authority and protected service meaning

## What this protects

Product purpose, policies, business rules, contracts, and the authority policy itself can be protected. An ordinary contributor may still write a proposal, failing test, implementation, issue, and PR. The change does not become approved product meaning until a configured product authority approves the exact commit.

Implementation code is not automatically protected. The restriction applies when a change alters what the service promises, permits, rejects, or requires.

## Assigning product authorities

Tell the AI which real Git accounts or teams may decide product direction:

```text
Only the product team and the Git account ryanmin may approve service policy changes.
```

Stackcord records the selected Git review provider, repository, allowed account subjects, protected kinds, minimum approvals, and whether an authority may approve their own change. Governance remains disabled until a real provider and account identity are selected, so a new solo project is not blocked by empty configuration.

Changing the authority list is protected by the current list. A contributor cannot add themselves and approve that same change. Git `user.name` and `user.email` are display metadata and never establish authority.

## Contributor and reviewer flow

```text
Contributor: Change the cancellation penalty.

Stackcord: This changes a business rule and contract. I can prepare it as a proposal,
including tests and implementation, but a product authority must approve the exact change.
I can create or update a PR and request the configured reviewer.
```

The AI runs `orchestrator governance check --json` before treating protected meaning as approved. If the current account is not an authority, it keeps authored meaning proposed and uses the one selected issue system for discussion or status. The PR or equivalent review owns approval of the actual change; an issue assignment or closed status does not.

After review, Stackcord refreshes the provider observation and checks the provider, repository, commit, protected fingerprint, review revision, approving account, and freshness. Any protected edit after approval makes that approval stale.

## What Git and the provider enforce

Repository rules such as CODEOWNERS, required reviewers, protected branches, or the selected provider's equivalent perform the actual merge restriction. Stackcord determines whether the diff changes protected service meaning and blocks integration or release when the exact approval cannot be verified.

Provider configuration is an explicit external write. Stackcord does not create a GitHub, GitLab, or other adapter before the user selects and connects that provider. It can use an available authenticated connector or CLI, explain the required repository rule, and then verify the normalized result.

## Clone and provider outage

The authority list and protected scope are committed, so another clone or AI recovers them. Live PR review evidence is local and ignored; Stackcord fetches it again from the selected provider. If the provider is unavailable, approval is unknown. Cached review data, commit names, comments, and issue status do not make it approved.

The default mode uses provider-account approval. Teams that need provider-independent cryptographic proof or multiple organization approvals can add signed approval requirements in the optional strict release profile.

## Important limitation

Stackcord cannot stop a person who controls the local filesystem from editing a file. It prevents an unapproved protected change from being recognized as canonical by its own checks and from passing integration or release. The Git provider's repository permissions and branch rules remain responsible for preventing unauthorized merges.
