# Service Continuity Harness — Production Product Design

**Status:** Final design selected from the critical product review

**Date:** 2026-07-18
**Supersedes:** `2026-07-17-focused-product-design.md` where the two conflict

## 1. Decision

Build a repo-native full-stack continuity harness for humans and AI working across one or more repositories.

The product starts with natural-language service discovery, preserves normalized product meaning, coordinates live work ownership, verifies actual Git and submodule state, detects semantic collisions before implementation, and binds exact workspace commits and evidence into one release candidate.

It is not a general AI-agent orchestrator, personal memory system, task tracker, specification methodology, code retrieval engine, build system, or developer portal.

The governing principle is:

> AI Skills own flexible conversation and product judgment. The cross-platform Go CLI owns deterministic state, safety, conflict, provenance, and identity verification.

## 2. Product promise

Plain-language promise:

> 여러 저장소에서 일하는 사람과 AI가 같은 서비스 규칙과 현재 작업 상태를 보고, 겹치는 변경을 시작 전에 찾고, 정확한 frontend/backend 조합으로 release하게 하는 프로젝트 하네스.

Public English claim, after it is proven by tests and dogfooding:

> Start or adopt a full-stack project with any stack, keep product contracts and live work aligned across repositories, prevent semantic collisions before implementation, and let the next human or AI recover the exact state needed to continue through release.

The product must not claim that it eliminates every conflict, gives an AI omniscience, supports every provider, or automatically publishes production systems.

## 3. Target users

Primary users:

- product teams of roughly 2–10 people with frontend, backend, worker, or service repositories;
- teams mixing people with multiple AI clients;
- consulting, handoff, or rotating-ownership teams;
- teams whose Jira or GitHub work status often drifts from contracts and actual code;
- teams that need business and failure behavior to remain consistent across components.

Secondary users:

- single-repository teams that still need business-contract and work-conflict coordination;
- larger organizations adding the optional strict-release profile to existing Jira, Backstage, and build systems.

The first target is not a solo toy project, a 20–50-agent autonomous swarm, a build-cache user, or an organization seeking a central SaaS developer portal.

## 4. Product boundaries

### 4.1 Default product

The default product owns:

1. natural-language service discovery and normalized checkpoints;
2. framework-neutral initialization and non-destructive adoption;
3. orchestration-root and child-workspace continuity;
4. combined actual-state recovery;
5. stable IDs, fingerprints, impact, stale, and unknown state;
6. product, business, behavior, interface, and data contracts;
7. executable work definitions and lifecycle;
8. one selected live task source and truthful provider reconciliation;
9. semantic conflict preflight and integration ordering;
10. safe Git, branch, worktree, and submodule operations;
11. canonical Git DBML and isolated dbdiagram proposals;
12. external UI source quarantine and authority;
13. TDD and evidence binding;
14. exact multi-workspace release identity;
15. Codex hooks, repo-local Skill, and Markdown fallback;
16. just-in-time technology and external-tool selection;
17. macOS and Windows Go binaries and Korean/English user guidance.

### 4.2 Optional strict-release profile

Strict release adds, without changing ordinary development:

- SBOM, provenance, and signature requirements;
- stronger supply-chain validation;
- organization-specific multi-approval and retention evidence;
- immutable staged publication policy;
- long-lived release branches or promotion gates when explicitly selected.

### 4.3 External systems

The product integrates with rather than reimplements:

- Superpowers for brainstorming, TDD, debugging, review, and execution discipline;
- one of BMAD, Spec Kit, or OpenSpec when deeper planning or SDD is selected;
- one of GitHub Issues, Jira, Beads, or Git-local for live work status;
- native or third-party Memory for personal and episodic recall;
- Gas Town for large-scale agent dispatch, mail, monitoring, and merge queues;
- Prune or code intelligence for source indexing and retrieval packs;
- Nx, Turborepo, or the chosen build system for task graphs and caches;
- Backstage for an organization-wide software catalog;
- Figma, Penpot, or local mockups for UI sources;
- dbdiagram for visual database collaboration.

### 4.4 Removed scope

Do not ship:

- fake provider adapters;
- an agent swarm, mailbox, daemon, dashboard, watchdog, or merge queue;
- a generic vector store, code index, or personal memory database;
- a task tracker, build cache, or service catalog;
- duplicated methodology Skills;
- operation-ID, receipt, or claim-file management as user UX;
- enterprise publication gates in the default flow;
- speculative framework and provider abstractions without an executable consumer.

## 5. User experience

The user speaks naturally:

- “이 서비스를 같이 만들어줘.”
- “이 프로젝트 이어서 해.”
- “지금 뭐 해야 해?”
- “계정 복구 기능 시작해줘.”
- “DB 변경을 같이 검토해줘.”
- “Production 준비가 됐는지 확인해줘.”

The user does not need to memorize CLI commands, stable IDs, operation IDs, provider snapshots, claim formats, or `.harness` paths. Skills invoke deterministic checks and translate results into confirmed facts, risks, and one safe next action.

Internal nouns appear only when they materially explain a failure.

### 5.1 Product flow without waterfall

The lifecycle is a navigation map, not a stage gate:

```text
entry diagnosis → service discovery → harness initialization
→ whole-product role/journey/UI coverage
→ small UI slice and shared contracts/data boundary
→ frontend/backend TDD implementation
→ continuous integration and production hardening
→ exact RC → technical and user validation → release → operation feedback
```

The whole-product coverage map is established before deep feature implementation so omitted roles, screens, failure states, and backend obligations remain visible. A large UI baseline is delivered in small role, domain, and journey slices instead of one long design phase. Each approved slice can proceed through UI, contract/data, implementation, and integration while discovery continues elsewhere.

An external mockup may supply the UI baseline. Its authority is registered before it drives implementation.

## 6. User-facing Skills

The Codex Plugin exposes five non-overlapping Skills.

1. `start-project`
   - discovers or imports service meaning;
   - writes normalized checkpoints after material answers;
   - initializes a new harness or adopts an existing project;
   - decides workspace boundaries without choosing a framework prematurely.

2. `continue-project`
   - performs combined status recovery;
   - locates the orchestration root from a child repository;
   - reports confirmed, stale, unknown, local-only, and blocked state;
   - recommends one safe next action.

3. `plan-project-work`
   - turns an approved slice into an executable work definition;
   - defines outcome, acceptance, semantic scope, dependencies, merge order, and evidence;
   - selects or creates the provider item;
   - prepares TDD work.

4. `coordinate-project-work`
   - claims work through the selected live provider;
   - creates safe branches or worktrees;
   - coordinates contracts, DBML, UI, migrations, and cross-repository integration;
   - resolves semantic conflicts and ownership changes.

5. `recover-and-release-project`
   - audits context after compaction or repeated questions;
   - records a handoff only when ownership changes;
   - performs production hardening and exact-RC preparation;
   - binds technical and user validation to the same candidate;
   - applies strict release only when enabled.

Every mutating Skill starts with the same recovery preflight. A separate overlapping “memory” or “audit” Skill is not added.

Generated repositories contain one compact repo-local entry Skill that routes the same five natural-language intents. Plugin-less continuity must not depend on global installation.

## 7. Repository architecture

```text
service-orchestration/
├── AGENTS.md
├── README.md
├── specs/
│   ├── index.md
│   ├── product/
│   ├── roles/
│   ├── journeys/
│   ├── decisions/
│   └── open-questions/
├── contracts/
│   ├── registry.yaml
│   ├── product/
│   ├── business/
│   ├── behaviors/
│   ├── interfaces/
│   └── data/
├── .harness/
│   ├── project.yaml
│   ├── profile.yaml
│   ├── sources.yaml
│   ├── workspaces.yaml
│   ├── work/
│   │   ├── definitions/
│   │   ├── mappings/
│   │   └── evidence/
│   ├── context/
│   ├── integration/
│   ├── release/
│   └── local/
├── frontend/
└── backend/
```

`frontend/` and `backend/` are ordinary directories in a single-repository project, monorepo projects, or submodules when independent repository ownership, release, permission, or deployment justifies it.

Submodules are a strong supported topology, not a mandatory topology.

The root is created or adopted once the service identity and repository boundary can be stated safely. A child repository and submodule are created as soon as an independent ownership, permission, deployment, or release boundary is approved; they are not postponed until the entire product is specified.

### 7.1 Shared versus local state

Committed canonical state:

- project and workspace identities;
- specs and contracts;
- work definitions and provider mappings;
- approved integration and release evidence references.

Selected provider state:

- live status, owner, comments, dependencies, and provider-native claim;
- provider revision or equivalent concurrency token.

Gitignored `.harness/local/` state:

- fetched provider snapshots;
- generated impact and context caches;
- compact context packets;
- machine and worktree observations;
- temporary import and comparison files.

Credentials never belong in any project state. Generated caches are not canonical and are rebuilt after clone. Hooks never mutate tracked files.

### 7.2 Root and child bridge

The orchestration root owns service-wide context, contracts, work, integration order, and exact child pointers. Each child owns its code and local build/test rules.

A child repository contains only a small bridge:

- project stable ID;
- orchestration repository URL and discovery method;
- child workspace ID;
- expected contract-set fingerprint;
- path to child build/test commands.

The CLI first uses actual Git superproject relationships, then the bridge. A standalone child clone cannot claim full service context; it requests the root clone or an explicitly approved contract snapshot.

## 8. Source hierarchy and context recovery

Recovery precedence is fixed:

1. actual root/child Git, remote, submodule, worktree, and selected-provider live state;
2. approved specs and contracts;
3. work definitions, mappings, claims, and verified evidence;
4. regenerated fingerprinted indexes and context packets;
5. chat summary, AI inference, and personal memory.

Lower-priority information never overrides higher-priority evidence.

### 8.1 Progressive context

The agent does not load the whole service into every prompt.

- Level 1: purpose, current focus, workspaces, active work, blockers, and open questions.
- Level 2: task-related role, journey, policy, contract, data, UI, dependency, and owner.
- Level 3: exact source documents, code, and tests required for the decision or edit.

Stable IDs and the impact graph select Levels 2 and 3.

### 8.2 Combined status

One status/resume operation checks:

- project and canonical-context identity;
- coverage gaps, decisions, assumptions, and open questions;
- root and child remotes, branches, HEADs, dirty state, upstream, and divergence;
- expected and actual submodule pointers;
- worktrees and branches already checked out;
- provider access, item revision, owner, status, and mapping drift;
- active semantic scope and observable claim state;
- stale or unknown contract, DB, UI, and generated artifacts;
- verified evidence and release-candidate state;
- one safe next action.

It distinguishes `confirmed`, `warning`, `stale`, `unknown`, `local-only`, and `blocked`. Cached external state cannot produce `confirmed`.

### 8.3 Compaction and new sessions

- `PostCompact` performs read-only status and injects a small packet through the correct Codex hook response.
- `SessionStart` locates the root and injects current project identity and next-action guidance.
- A repo-local Skill performs the same preflight when hooks are unavailable.
- Markdown fallback states the reduced verification level on unsupported clients.

Memory can accelerate lookup, but a memory result is an unverified hint until its source and fingerprint are checked.

### 8.4 Remote recoverability

Uncommitted files, unpushed commits, local-only worktrees, and local-only memory cannot be recovered on another computer. The product reports this plainly and never hides a push.

Shared work should have a verified small commit, a remote branch or review, live provider owner/status, the current work-definition fingerprint, and the next failing test or action.

## 9. Service discovery and coverage

The default discovery model is free conversation plus coverage-driven questions and immediate normalized checkpoints.

The AI first reads repository facts. It asks only questions that materially change the product, architecture, current slice, security, or operation. Questions are one at a time, recommend one option first, provide two or three meaningful alternatives, and allow free input.

After each material answer the product stores only:

- product facts and decisions;
- new or changed contracts;
- success, failure, duplicate, retry, timeout, cancellation, and compensation scenarios;
- unresolved questions and their re-evaluation conditions;
- affected UI, data, interface, workspace, and work scope;
- source, approval time, stable ID, and fingerprint.

Raw conversation, tone, prompts, and credentials are not stored.

Coverage is tracked across roles, journeys, value and non-goals, business rules, failure and recovery, authorization and security, privacy and data lifecycle, accessibility, operations, support, UI, contracts, and data. Each area is `known`, `assumed`, `open`, or `stale`.

There is no fixed questionnaire count and no “finish 200 questions before code” gate. Critical gaps block the affected slice; non-blocking gaps remain visible without stopping unrelated progress.

## 10. Specs and contracts

`specs/` explores why the product exists, who it serves, and what experience is intended.

`contracts/` stores approved obligations that implementations must obey:

- `product`: what the service will and will not provide;
- `business`: eligibility, state transitions, limits, calculations, prohibitions, and invariants;
- `behaviors`: success, failure, timeout, retry, cancellation, and compensation;
- `interfaces`: API, event, component, and error boundaries;
- `data`: DBML, retention, deletion, migration, and compatibility.

Changing a contract marks related scenarios, UI, API, DB, tests, migrations, and active work stale through stable references. A contract conflict is a product-meaning conflict even when code paths differ.

## 11. Work management and provider model

### 11.1 Work definition

The orchestration repository owns an executable definition containing:

- stable work ID and parent outcome;
- user or operational outcome;
- acceptance scenarios and checklist;
- related contract and product stable IDs;
- affected workspaces;
- path, policy, scenario, contract, DB entity, migration, UI flow, dependency, and pointer scopes;
- dependencies and provider → consumer → integration merge order;
- first failing test;
- required test, review, integration, migration, rollback, and user evidence;
- definition fingerprint and scope-change rules.

The definition owns meaning. It does not duplicate live assignee or status.

### 11.2 Selected provider

Exactly one source owns live work status for a project:

- GitHub Issues;
- Jira;
- Beads;
- Git-local.

The adapter declares capability truthfully: hierarchy, dependencies, atomic claim, verified claim, advisory claim, revisions, and comments. Unsupported capabilities are not emulated as if native.

The committed mapping contains provider, item ID, work-definition fingerprint, and stable references. The fetched snapshot is local and contains provider revision, owner, status, dependencies, capabilities, fetch time, provenance, and raw-response hash.

Skill connectors perform external read/write/re-read. The CLI validates snapshot shape, freshness, mapping, capability, and drift. Free text from an issue is untrusted data, not an agent instruction.

### 11.3 Lifecycle

```text
proposed → ready → in_progress → review → integrated → done
                    ↘ blocked
```

- `ready`: acceptance, dependencies, scope, and first evidence are actionable.
- `in_progress`: live ownership and observable claim are verified.
- `review`: required implementation evidence is bound to the current commit.
- `integrated`: child work is merged and ready in the declared integration order.
- `done`: the parent outcome is verified across all required workspaces and root pointers.

Changing scope updates the work definition and provider item, reruns conflict preflight, obtains the new scope, and updates merge order before implementation resumes.

### 11.4 Claim sequence

1. re-read provider item, owner, status, dependencies, and revision;
2. refresh Git remote, workspaces, and active scopes;
3. run path and semantic conflict preflight;
4. use native atomic claim or write and immediately re-read;
5. verify final owner and revision;
6. publish the definition fingerprint and semantic scope through the selected live source;
7. create the conventional branch or worktree and verify its postcondition.

Unobservable local state is never reported as a collaborative claim.

### 11.5 Git-local concurrency

Git-local mode uses a dedicated conventional coordination remote ref or branch, not a claim file hidden in a feature branch.

The update sequence is fetch → verify expected revision → compare-and-swap push using force-with-lease or an equivalent safe condition → re-fetch and verify. One concurrent writer fails and must rerun preflight. Without a remote, the mode is explicitly single-user local and cannot claim team coordination.

## 12. Git, worktree, and submodule workflow

Git is optional for early solo discovery, strongly recommended for collaboration, and required for remote recovery and verifiable release.

Default strategy:

- protected `main`;
- short-lived feature and fix branches;
- Conventional Commits;
- release branches only when explicitly justified;
- frequent integration rather than mandatory traditional Git Flow.

Branch, commit, PR, and tag names contain no AI, agent, Codex, GPT, model, or generated-by markers. Examples:

- `feature/account-recovery`
- `feature/ABC-142-account-recovery`
- `fix/payment-timeout`
- `feat(account): validate recovery challenge`

### 12.1 Safe mutation

Read-only checks run automatically. Reversible local changes inside the requested workflow may run after preconditions. External writes run only when the user requested or project policy authorizes that workflow. Destructive overwrite, credential creation, public publication, and other irreversible external actions require separate authority.

The product never silently pulls, rebases, stashes, resets, cleans, force-pushes feature history, or overwrites dirty or diverged state.

### 12.2 Worktrees

Worktrees isolate concurrent local branches and dirty files. They do not replace shared claims. Before creation the CLI verifies base revision, branch naming, duplicate checkout, location, and repository identity; afterwards it verifies path, branch, and HEAD.

Each root worktree initializes and verifies its own submodule state. A child HEAD in another worktree cannot satisfy the current root pointer.

### 12.3 Submodules

Submodules preserve the exact child commit combination; they are not locks. Detached child HEAD can be a normal root-pinned state. A conventional branch or child worktree is created before child development.

Root pointers change only at verified integration checkpoints, not for every child commit. Pointer updates use an integration queue and declared merge order to avoid repeated gitlink collisions.

Compatibility order is additive contract → provider → consumer/mock → UI connection → migration/rollback where applicable → exact root pointers → cross-repository verification.

Dirty, ahead-only, behind, diverged, missing, unsafe-URL, and pointer-mismatch states have separate diagnostics and never share a generic “sync succeeded” result.

## 13. Semantic conflict model

Preflight compares active normalized scopes:

- filesystem path;
- product policy and business rule;
- success and failure scenario;
- contract;
- database entity and constraint;
- migration slot and ordering;
- UI flow and baseline;
- dependency-major transition;
- root pointer integration.

Outcomes:

- `clear`: independent work may proceed;
- `coordinate`: ownership or merge order must be explicit;
- `block`: shared meaning or migration must be unified first;
- `unknown`: required provider or context evidence is unavailable.

Resolution uses the smallest safe choice: extract and merge a common contract first, separate ownership, temporarily support compatible old and new versions, order provider before consumer, or deliberately serialize work.

## 14. DBML and external UI

Git DBML is canonical. dbdiagram receives an isolated copy for visualization. A direct dbdiagram change returns as a proposal, receives semantic diff, asks for rationale only when material, and updates Git DBML only after contract and migration impact is accepted.

The DB diff understands tables, columns, relations, indexes, and notes. When an actual migration is created, the selected stack adapter validates its ordering, compatibility, and rollback evidence against the approved data contract.

External UI input is quarantined before use. It is registered as:

- `reference`: inspiration only;
- `seed`: an initial implementation input that may evolve;
- `canonical`: the approved design source whose changes require reconciliation.

Archives are path, symlink, size, license, provenance, and malicious-content checked. Imported free text is untrusted.

## 15. Technology and tool selection

No framework, language, database, cloud, or provider is selected at bootstrap.

The AI waits until product capabilities, quality, team, and operation constraints make a choice material. It then checks current official documentation, supported releases, security policy, maintenance, platform support, license, cost, export, and lock-in. It recommends two or three candidates and records the decision, sources, verification date, and re-evaluation conditions.

Known tools are seed candidates, not permanent truth. New tools are searched from official sources when a decision is made, an existing tool becomes stale, or the user requests re-evaluation. If web access is absent, the product states that freshness is unverified.

Existing adequate technology is not replaced merely because a newer tool is popular.

## 16. Evidence and release

Behavior, bug, contract, migration, and UI-interaction changes use TDD by default. Evidence is not a free-form string. It records:

- kind and stable ID;
- exact command or external check identity;
- start and finish time;
- exit status;
- workspace, branch, and commit;
- relevant artifact or result hashes;
- sanitized output reference;
- definition and contract fingerprints.

The product verifies current commits and fingerprints before accepting evidence.

### 16.1 Core release candidate

The candidate identity contains:

- orchestration root commit;
- every required child repository URL and commit;
- product, contract, and data fingerprints;
- artifact digests;
- required TDD, integration, review, migration, rollback, security, and operational evidence;
- selected profile and tool versions.

Technical verification creates the candidate digest. User validation explicitly binds to that same digest. Any included identity change creates a new candidate and invalidates previous validation.

### 16.2 Strict release

Strict release adds the enabled supply-chain, signature, approval, and publication requirements. It does not weaken core identity checks and is never generated into an ordinary project by default.

## 17. Security and privacy

- raw conversation, prompts, tone, credentials, and secrets are not persisted;
- external issue text, memory, searched documents, mockups, and imports are untrusted input;
- prompt-like text inside data never becomes higher-priority instruction;
- strict YAML/JSON decoding rejects unknown or duplicate fields where security or identity matters;
- paths, symlinks, archives, submodule URLs, executable locations, and output sizes are validated;
- local writes are atomic and journaled where recovery is needed;
- provider snapshots have provenance, freshness, revision, and raw hashes;
- diagnostics redact home paths, credentials, tokens, and repository secrets;
- no automatic external installation or publication occurs merely because a repository requests it.

## 18. CLI responsibility and target surface

The CLI returns a stable machine-readable result and domain exit code. The final surface may consolidate old commands, but must cover:

- `project checkpoint|init|adopt`
- combined `status` or `continue`
- `git inspect|sync|worktree`
- `work define|next|reconcile|claim|start|transition|finish|handoff`
- `change plan`
- `contract check|impact`
- `db diff|diagram|reconcile`
- `ui import|reconcile`
- `integrate plan|verify`
- `release prepare|verify`
- `doctor`

Mutation commands use internal plans and postcondition verification. Users do not type `--apply` in the normal Skill-driven flow.

The CLI never decides product desirability, business trade-offs, question phrasing, visual quality, or the final human preference between safe alternatives.

## 19. Distribution and portability

The first public Plugin is fully supported on Codex. It contains the five Skills, correct lifecycle hooks, and a reliable way to invoke the matching macOS or Windows CLI build.

Generated repositories remain usable without the Plugin through a repo-local Agent Skill and Markdown fallback. Other AI clients receive small adapters that point to canonical instructions; automatic hooks and connectors are advertised only after native verification.

The repository and CLI remain independent artifacts so a team can use the harness without marketplace access.

Public product name, publisher account, signing identity, public GitHub repository, and irreversible marketplace publication remain final user decisions.

## 20. Migration of the current implementation

Keep and strengthen:

- normalized checkpoints and non-destructive generation;
- stable IDs, fingerprints, and impact graph;
- real Git, worktree, and submodule inspection;
- semantic conflict scopes;
- contract, DBML, UI quarantine, atomic operations, and release digest cores;
- five existing user-facing Skill names;
- bilingual output and strict-release isolation.

Replace or complete:

- the document-only context audit with combined status;
- shallow WorkItem and advisory local claim with the work lifecycle and verified providers;
- plan-only branch, worktree, and submodule operations with safe apply and postconditions;
- string-only evidence with commit-bound execution evidence;
- fixed integration text with work-definition-driven integration;
- manual release input with actual-state collection and verification;
- incorrect hooks and validators with current Codex schemas;
- uncertain CLI installation with verified cross-platform bootstrap;
- string-presence Skill tests with agent behavior evaluations.

Remove or leave strict:

- unsupported generic provider claims;
- duplicate policy and lifecycle documents;
- default supply-chain publication requirements;
- user-facing operation and receipt mechanics;
- speculative platform abstractions.

The preserved historical branch remains the recovery point for removed broad work. No existing implementation is deleted before replacement tests define the required behavior.

## 21. Production acceptance

Production completion requires evidence for all of the following:

1. new service discovery, repeated checkpoint, and framework-neutral initialization;
2. non-destructive adoption of an existing project;
3. actual orchestration root plus frontend/backend Git repositories and submodules;
4. clean-clone, new-session, post-compaction, different-AI, and plugin-less recovery;
5. actual dirty, ahead, behind, diverged, detached, missing, local-only, unsafe, and pointer-mismatch handling;
6. complete work definition, lifecycle, provider mapping, claim race, scope expansion, and handoff;
7. Git-local compare-and-swap plus GitHub, Jira, and Beads connector snapshot/re-read/drift behavior;
8. cross-repository semantic conflict without path overlap;
9. contract-first frontend mock and backend provider TDD, followed by exact root integration;
10. DBML/dbdiagram proposal reconciliation and migration-order conflict;
11. external UI reference, seed, and canonical flows with malicious archives;
12. commit-bound test, review, integration, migration, rollback, and user evidence;
13. exact core RC and optional strict-release verification;
14. macOS and Windows builds, Go race, fuzz smoke, static analysis, vulnerability and secret scans;
15. official Plugin and Skill validation, correct Hook behavior, and plugin-less fallback;
16. Korean/English README, five-minute quickstart, demo transcript, provider/tool guide, and troubleshooting;
17. dogfooding report and baseline comparison for recovery time, repeated questions, false claims, missed semantic conflicts, wrong-repository edits, pointer drift, and invalid RC combinations.

Tests and documentation are evidence only when they execute the stated scenario. String-presence tests do not prove agent behavior, and a fixture-only provider test does not prove a real external write.

## 22. Failure behavior

- Unknown external state is `unknown`, not success.
- A stale work-definition mapping requires reconciliation before claim.
- A failed compare-and-swap claim causes refresh and preflight, not silent retry with overwrite.
- Dirty or diverged work blocks pointer and branch mutation.
- A missing root makes child context incomplete.
- A failed Hook falls back to Skill preflight without claiming automatic recovery.
- A missing CLI enables documented reduced verification only.
- Changed release identities invalidate the previous candidate and user validation.
- A material unanswered coverage gap blocks only the affected slice.

Every failure returns the relevant evidence and one safe next action without exposing internal mechanics unnecessarily.

## 23. Official ecosystem references reviewed

- Superpowers: https://github.com/obra/superpowers
- BMAD Method: https://docs.bmad-method.org/reference/workflow-map/
- Spec Kit: https://github.github.com/spec-kit/
- OpenSpec: https://github.com/Fission-AI/OpenSpec
- Beads: https://github.com/gastownhall/beads
- Gas Town: https://github.com/gastownhall/gastown
- Prune: https://prune.codes/docs
- Nx affected/project graph: https://nx.dev/docs/features/ci-features/affected
- Backstage Software Catalog: https://backstage.io/docs/features/software-catalog/
- Codex agent loop and compaction: https://openai.com/index/unrolling-the-codex-agent-loop/
- Codex memory implementation notes: https://github.com/openai/codex/blob/main/codex-rs/core/src/memories/README.md
- GitHub custom agents: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/about-custom-agents

These links record what was reviewed on 2026-07-18. Tool selection in a real project rechecks current official sources rather than treating this list as permanent truth.
