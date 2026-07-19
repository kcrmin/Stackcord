# Product Governance — Account-Based Approval Design

## Decision

Stackcord protects changes to product purpose, policy, business rules, and contracts without preventing ordinary contributors from proposing or implementing them. A contributor may create a proposal, tests, and implementation, but protected product meaning becomes approved only after a configured product authority approves the exact change.

The default proof is an approval by an allowed account in the selected Git review provider. Git `user.name` and `user.email` are display metadata and never prove authority. Teams that need provider-independent or offline verification may additionally require a signed Git approval under the optional strict profile.

## Goals

- Let a team assign the people or Git teams allowed to approve service direction.
- Keep ordinary implementation work lightweight.
- Allow any contributor to propose a protected change without silently making it canonical.
- Bind approval to the exact protected-meaning fingerprint, not a ticket title or mutable branch name.
- Preserve enough committed policy to recover authorization rules after clone or context compaction.
- Use the selected Git provider for live review evidence without making GitHub, GitLab, or another provider mandatory.
- Block integration and release when approval is missing, stale, forged, or unavailable.

## Non-goals

- Preventing a user who controls the local filesystem from editing a file.
- Treating Git author or committer names and email addresses as authentication.
- Replacing repository permissions, protected branches, CODEOWNERS, or provider review rules.
- Building speculative adapters for providers the user did not select.
- Requiring signed commits or organization-grade approval receipts in ordinary projects.

## Alternatives considered

### Agent instructions only

An instruction can remind an AI to request approval, but a contributor or another tool can bypass it. This is documentation, not enforcement, so it is rejected as the product guarantee.

### Git-provider account approval plus Stackcord verification — selected

The project commits who may approve and which semantic areas are protected. The selected Git provider enforces protected review, while Stackcord compares the live reviewer identity and revision with the exact protected-change fingerprint before integration or release. This fits normal team workflows and keeps the default mode usable.

### Signed approval records for every protected change

SSH or GPG signatures can remain verifiable after clone without the provider, but key setup and rotation add friction. This remains an optional stronger policy rather than the default.

## Canonical governance policy

The orchestration root stores a versioned governance policy. Users normally manage it through natural-language requests rather than editing it directly.

```yaml
schema_version: 1
product_authorities:
  - provider: github
    subject: user:ryanmin
  - provider: github
    subject: team:product
protected_kinds:
  - product
  - policy
  - business
  - contract
approval:
  minimum: 1
  authority_self_approval: true
  signed_git: optional
```

Provider and subject values are normalized identities returned by the selected provider adapter. A provider-neutral project may declare identities only after choosing a real provider. Stackcord does not invent or claim support for an unavailable provider.

Changing the governance policy is itself protected. The first project initializer becomes the bootstrap authority only after an explicit identity selection. After bootstrap, adding or removing an authority requires approval under the current policy, so a contributor cannot grant themselves authority in the same change.

An existing authority may approve their own protected change by default because a solo product owner must remain able to work. Teams that require separation of duties can disable `authority_self_approval` and require another allowed account. Self-approval never lets an account outside the current approved authority set grant itself access.

## Protected change model

Protected meaning includes:

- service purpose and non-goals;
- product and service policy;
- business rules and authorization behavior;
- product, business, behavior, interface, and data contracts;
- governance policy itself.

Implementation files are not protected merely because they are code. Stackcord follows stable references from a changed spec or contract into UI, API, database, migration, tests, and active work. A contributor may work on those dependents, but the protected source remains `proposed` until approved.

Each approval is bound to:

- repository identity;
- protected stable IDs and revisions;
- protected-meaning fingerprint;
- exact head commit reviewed;
- provider and immutable review revision;
- normalized approving account;
- observation time and source provenance.

Branch names, issue status, assignment, comments, Git author names, and cached review snapshots are not approval evidence.

An authored `status: approved` field is also not proof by itself. Effective approval is calculated from the current governance policy and exact provider or signed-Git evidence. This avoids a second bookkeeping commit after review while preventing a contributor from declaring their own proposal approved in a file.

## User flow

### Contributor proposes a service change

1. The AI detects that the request changes protected product meaning.
2. Stackcord identifies the current Git account through the selected provider, not local Git display metadata.
3. If the account is not a product authority, the AI explains that it can prepare a proposal but cannot approve it.
4. The proposal, affected contracts, scenarios, tests, and implementation plan are connected to one work item and ordinary branch.
5. The provider receives a PR or equivalent review request with the configured authority as required reviewer.
6. Integration and release remain blocked while the protected change is proposed or approval evidence is unknown.

### Product authority approves

1. Stackcord refreshes the exact provider review and rejects cached or mismatched evidence.
2. It verifies that an allowed account approved the exact reviewed commit and protected-meaning fingerprint.
3. It marks the protected revision approved and refreshes affected context, contracts, work, and integration order.
4. If any protected source changes afterward, the approval becomes stale and must be repeated.

When the current actor is already an allowed product authority, Stackcord may accept that authority's provider-authenticated merge or configured signed Git action as approval. A team that disables authority self-approval still requires a separate allowed reviewer.

### Clone, provider outage, and AI context recovery

The committed governance policy and protected revisions recover from Git. Live account approval is refreshed from the selected provider. If the provider cannot be reached, Stackcord reports that approval cannot be verified and does not guess from commit metadata. A valid optional signed approval may satisfy the configured offline policy.

## Provider and Git enforcement

Stackcord generates or validates the selected provider's equivalent of required owners, protected branches, and required reviews only after that provider is chosen. For GitHub this may include CODEOWNERS and rulesets or branch protection; other providers use their supported equivalent. Provider configuration changes are explicit external writes.

Stackcord remains a second verification layer:

- the Git provider controls who may approve and merge;
- Stackcord understands whether the change affects protected product meaning;
- the release gate requires fresh approval evidence for every selected protected revision.

Neither layer silently substitutes for the other.

## CLI and Skill behavior

The deterministic CLI will:

- validate governance policy syntax and bootstrap invariants;
- calculate the protected-change fingerprint;
- report whether the current actor may approve or only propose;
- verify normalized provider approval observations;
- mark approval stale when the commit, stable IDs, revision, or fingerprint changes;
- block integration and core release when selected protected work lacks fresh approval;
- keep provider payloads, credentials, and local observations out of committed canonical files.

The Skill will:

- infer that a request changes protected meaning;
- explain the restriction in ordinary language;
- offer to prepare a proposal and review request;
- never claim that local Git name or email proves authority;
- never expose internal approval IDs or storage paths during normal use;
- request an external write only when the user asked to create or update the review.

## Failure behavior

- Unknown current account: proposal is allowed; approval and integration are blocked.
- Unauthorized approval: rejected and recorded as a validation failure.
- Approval for an older commit or fingerprint: stale and rejected.
- Provider unavailable: status is unknown; cached approval is not accepted as live.
- Governance file changed without current-authority approval: blocked.
- Authority removed by an approved governance change during active review: refresh policy and require an authority allowed by the current approved policy.
- Insufficient required approvals: protected meaning remains proposed.
- Local Git author matches an authority but provider identity does not: unauthorized.

## Security and privacy

Only normalized account identifiers, review revision, timestamps, and hashes enter local observations or release evidence. Raw provider comments and payloads remain untrusted and are not executed. Tokens stay in the selected provider client or connector. Governance changes, provider observations, and signed approvals use the existing path, schema, symlink, size, and secret checks.

## Testing

Deterministic tests cover:

- schema acceptance and rejection;
- bootstrap authority and self-escalation prevention;
- contributor proposal versus authority approval;
- spoofed Git name and email rejection;
- exact commit and protected-fingerprint matching;
- stale approval after protected meaning changes;
- unavailable provider and cached observation behavior;
- integration and core-release blocking;
- optional signed approval behavior;
- clone and multi-repository recovery;
- generated project, repo-local Skill, Markdown fallback, README, and schema parity.

No live provider or Codex call runs in ordinary unit tests or PR checks. Provider behavior uses normalized deterministic fixtures. A real provider scenario is manual and runs only after that provider is explicitly selected and configured.

## Documentation promise

README wording must distinguish proposal from approval and must not claim that Stackcord can prevent arbitrary local edits. It explains that repository access controls perform the actual merge restriction while Stackcord detects protected product meaning, verifies the exact approval, and blocks unapproved integration or release.
