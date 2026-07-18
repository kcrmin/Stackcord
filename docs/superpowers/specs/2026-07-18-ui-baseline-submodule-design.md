# Editable UI Baseline Workspace — Product Design

**Status:** Approved design

**Date:** 2026-07-18
**Extends:** `2026-07-18-service-continuity-harness-design.md`

## 1. Decision

Add `ui/` as an optional first-class workspace and supported Git submodule boundary.

`ui/` owns the editable product UI baseline: user journeys, screens, states, interactions, design tokens, accessibility expectations, approved assets, prototypes when selected, and the provenance of imported sources. `frontend/` remains the production implementation and consumes an exact approved `ui/` commit together with applicable product and technical contracts.

The UI workspace is framework-neutral by default. A prototype framework, design application, or UI-generation Skill is selected only when the project needs it.

The governing rule is:

> External tools help create and understand the UI. The committed `ui/` baseline defines what the product intends to build.

## 2. Problem

The existing product can register external UI sources and coordinate UI-flow conflicts, but an imported mockup currently remains mostly an orchestration-side source record. That is insufficient when a team wants to:

- keep UI design and production frontend in separate repositories;
- import an existing mockup and continue editing it;
- build a new UI baseline before choosing a frontend framework;
- use Figma exports, HTML, images, videos, or UI Skills as starting material;
- bind frontend work to an exact UI decision rather than a changing folder;
- let another clone recover which UI version the frontend implements;
- detect a changed UI baseline before integrating stale frontend work.

The product must support those workflows without becoming a design tool, a frontend framework, or a generic artifact repository.

## 3. Product boundary

### 3.1 The orchestration root owns

- service meaning, business rules, policies, and shared contracts;
- workspace identities and dependency relationships;
- the expected root pointer for each submodule;
- UI source registration, authority, provenance, and impact relationships;
- work ownership, semantic conflict checks, integration order, and release identity.

### 3.2 The `ui/` workspace owns

- role- and journey-based UI coverage;
- screen composition and navigation intent;
- loading, empty, error, success, permission, offline, and destructive-action states;
- responsive and accessibility expectations;
- interaction behavior that is independent of implementation framework;
- shared visual tokens and approved assets;
- editable imported material and optional prototypes;
- source attribution, license, version, hash, and selection rationale.

### 3.3 The `frontend/` workspace owns

- production code and runtime behavior;
- framework-specific components and state management;
- browser, device, performance, and accessibility implementation;
- tests proving the selected UI baseline and contracts are implemented;
- evidence tied to the exact UI baseline commit.

### 3.4 This feature does not own

- a Figma, Penpot, or browser-based design editor;
- automatic conversion of any mockup into production-quality frontend code;
- a mandatory prototype framework;
- a mandatory external Skill collection;
- binary asset versioning infrastructure beyond ordinary Git policy;
- remote repository creation for every Git provider.

## 4. Workspace topology

The recommended multi-repository topology is:

```text
service-orchestration/
├── specs/
├── contracts/
├── .harness/
├── ui/                         # optional UI baseline submodule
│   ├── README.md
│   ├── coverage/
│   ├── flows/
│   ├── screens/
│   ├── states/
│   ├── tokens/
│   ├── accessibility/
│   ├── assets/
│   ├── prototypes/             # only when the team selects one
│   └── sources/
├── frontend/                   # production frontend submodule or directory
└── backend/                    # backend submodule or directory
```

`ui/`, `frontend/`, and `backend/` may also be ordinary directories in one repository. A submodule is recommended when independent ownership, permissions, history, release, or deployment make the boundary useful. The CLI must not force submodules merely because the folder names exist.

The workspace graph records the dependency:

```text
frontend -> ui
frontend -> applicable contracts
backend  -> applicable contracts
root     -> exact ui/frontend/backend commits
```

## 5. User flows

### 5.1 No UI baseline exists

1. The user explains the service, roles, journeys, constraints, and desired experience in natural language.
2. The Skill continuously writes normalized product decisions and unresolved questions.
3. The Skill proposes an optional UI creation method only when it becomes useful.
4. The user may select a UI Skill collection, a design tool, an external designer, or direct repository editing.
5. Candidate work is placed in or promoted into the `ui/` workspace.
6. The team completes missing states, accessibility, responsiveness, and failure behavior.
7. The approved result is committed as a UI baseline.
8. Frontend work is planned against that exact commit and implemented with TDD.

### 5.2 A partial mockup or prototype exists

1. The Skill identifies its format, intended authority, license, and expected use.
2. The CLI safely inspects the import for traversal, links, executable content, secrets, excessive size, and unsupported structures.
3. The user and Skill compare it with the current UI baseline.
4. Selected material is promoted into the editable `ui/` workspace.
5. Existing work is preserved; conflicting product meaning is resolved explicitly.
6. Missing screens and behavioral states are added.
7. The resulting commit becomes the new baseline.

### 5.3 An approved external design exists

1. The source is registered as `canonical` after safety and provenance checks.
2. The entire approved material, or an export suitable for version control, is promoted into `ui/`.
3. The Skill checks for unrepresented error, loading, permission, destructive, responsive, and accessibility behavior.
4. The team edits and commits the imported material normally.
5. Frontend implementation references the exact approved UI commit.

### 5.4 Reference-only material exists

The source is registered as `reference`. It informs decisions but cannot silently replace the baseline or satisfy frontend acceptance. If the team later decides to adopt it, authority changes through an explicit reviewed promotion.

## 6. Import and promotion model

The phrase “do not copy external mockups directly into `ui/`” is not a product rule. The precise rule is:

> Untrusted external material is checked before it enters the canonical workspace. After review, accepted material can be copied into `ui/`, edited, reorganized, committed, and used as the product baseline.

The temporary quarantine is an internal safety boundary, not a workflow the user must manage.

### 6.1 Source authority

- `reference`: inspiration or comparison only;
- `seed`: an editable starting point that may be substantially changed;
- `canonical`: an externally approved source whose accepted version defines the current baseline.

Authority describes decision weight, not file immutability. Material imported as `canonical` can still be edited; doing so produces a new reviewed baseline with recorded provenance.

### 6.2 Promotion modes

- `whole`: copy the reviewed source as one editable source tree;
- `selected`: copy only reviewed files or regions chosen by the team;
- `reference-only`: retain provenance and comparison material without placing it in the editable baseline.

Promoted raw material is placed beneath `ui/sources/<source-id>/` unless it already conforms to the selected UI workspace structure. The Skill then normalizes product meaning into `flows/`, `screens/`, `states/`, `tokens/`, and `accessibility/`. This avoids pretending that an arbitrary export already expresses all required product behavior.

The complete original archive is preserved only when size, license, audit, or reproducibility needs justify it. A stable source record, content hash, selected authority, and rationale are always retained.

## 7. Optional UI creation tools

The product detects or presents tools only after examining the project and current UI material. It explains why a tool may help, its trade-offs, and whether it creates portable files or tool-specific state. Only the user's selected tool is connected.

Examples include:

- Figma, Penpot, exported HTML, images, video, or an external design team;
- design-to-code or prototype tools chosen at the time of use;
- Codex-compatible UI Skill collections such as MengTo/Skills;
- project-native component explorers or visual test tools after a frontend stack is selected.

MengTo/Skills-like workflows may help generate design directions, extract interactions from HTML or video, capture references, or produce prototypes. They do not own service truth, workspace identity, UI authority, or release evidence. Their outputs enter through the same source, promotion, review, and baseline process as any other external material.

The generated project remains usable without those tools. Repo-local Skills and the Markdown fallback explain the approved process even when the original global Plugin or optional UI Skills are absent.

## 8. Deterministic CLI responsibilities

The Skill owns conversation, interpretation, comparison, and recommendations. The CLI owns the operations that must be reproducible.

### 8.1 Safe submodule creation

Add a typed plan/apply operation for an existing remote repository. Before mutation it verifies:

- execution from the exact orchestration root;
- valid Git repository and expected active branch;
- no detached, diverged, or unsafe root state;
- clean tracked state required for a predictable submodule addition;
- a credential-free supported Git remote URL;
- an absent and safe target path;
- no conflicting `.gitmodules` declaration;
- bounded execution and explicit postconditions.

It never creates a remote provider repository, commits, pushes, or opens a pull request automatically. The Skill may guide the user through the selected provider separately.

### 8.2 Workspace registration

Add a typed plan/apply operation that:

- registers the workspace stable ID, kind, path, remote, responsibilities, and dependencies;
- supports `ui-baseline` as a responsibility without assuming a framework;
- writes the child bridge needed to recover the orchestration root;
- initializes the minimal UI baseline structure when requested;
- links a frontend workspace to its UI dependency;
- refuses destructive replacement of an existing workspace configuration.

Submodule mutation and workspace registration remain separate recoverable operations. A failure in one can be diagnosed and resumed without guessing what partially succeeded.

### 8.3 Source promotion and baseline binding

Extend UI operations to:

- inspect and register external sources;
- plan and apply `whole`, `selected`, or `reference-only` promotion;
- prevent paths escaping the selected UI workspace;
- preserve source identity and provenance;
- bind an approved baseline to a clean UI commit, remote identity, and root submodule pointer;
- reject local-only, dirty, unpublished, stale, or pointer-mismatched baselines when used for integration or RC.

The CLI copies only explicitly selected, previously inspected files. It does not decide which visual direction is better or silently overwrite canonical UI files.

### 8.4 Frontend dependency and staleness

A frontend work definition records the UI baseline identity and fingerprint it implements. The combined status and integration checks report:

- exact match;
- newer UI baseline available;
- referenced UI commit missing;
- local-only UI commit;
- root pointer mismatch;
- changed flow, state, token, or asset affecting active frontend work;
- unknown state when evidence cannot be proved.

The default UX explains the consequence and next action without exposing internal fingerprints unless debugging requires them.

## 9. Conflict prevention and integration

Before UI or frontend work begins, semantic scope can claim:

- roles and journeys;
- UI flows and screens;
- named UI states and interactions;
- design tokens and shared assets;
- product policies and business rules;
- interface contracts, DB entities, and migrations;
- workspace dependencies and submodule pointers.

Path overlap is only one signal. Two branches editing different files still conflict when they change the same checkout failure behavior, permission rule, entity meaning, or shared token.

When overlap is safe, work continues in independent worktrees and repositories. When order matters, the integration plan records it. Common sequences include:

```text
UI baseline -> frontend implementation
contract -> backend implementations -> merge -> frontend connection
migration -> backend deployment compatibility -> frontend activation
shared token change -> affected UI slices -> affected frontend components
```

The product does not force a single sequence for every feature. It chooses a sequence from the actual semantic and runtime dependency graph.

## 10. TDD and evidence

TDD remains the default for UI interactions, contracts, migrations, runtime behavior, and bugs.

For UI-to-frontend work, evidence may include:

- acceptance tests derived from the approved flow and states;
- component and interaction tests;
- accessibility checks;
- responsive visual or screenshot comparison when the selected stack supports it;
- contract tests for frontend/backend boundaries;
- exact UI baseline commit and source provenance;
- integration results from the root workspace.

A screenshot alone is not proof of interaction, failure behavior, or accessibility. A frontend test suite alone is not proof that the intended UI baseline was used. Release evidence binds both.

## 11. Clone and context recovery

After cloning the orchestration repository, the user can ask “이 프로젝트 이어서 해.” The Skill invokes combined recovery and reports:

1. whether required submodules are initialized;
2. actual root pointer versus each child HEAD;
3. dirty, ahead, behind, diverged, local-only, and missing-remote state;
4. the current approved UI baseline and source authority;
5. frontend work tied to an older or missing UI baseline;
6. active work ownership, semantic conflicts, and integration order;
7. the next safe action.

If context has been compressed or the AI repeats settled questions, context audit reconstructs the same facts from committed product summaries, decisions, contracts, work definitions, workspace bridges, Git state, and evidence. Raw conversation history is not required.

## 12. README experience

The Korean and English README files will be rewritten for fast scanning and parity. The top-level document should contain only:

1. one-sentence product description;
2. four concrete outcomes;
3. three natural-language prompts to start, adopt, or continue;
4. one small diagram showing discovery, `ui/`, frontend/backend, integration, and RC;
5. a short new/adopt/continue path;
6. a compact feature table;
7. a five-minute local installation and first-use path;
8. links to detailed submodule, UI, task, DBML, release, security, and troubleshooting guides;
9. an accurate Codex-first support statement;
10. a clear distinction between default and strict release.

Advanced internal terminology, command catalogs, schema detail, and enterprise release options belong in linked guides rather than the README opening.

## 13. Naming

The public product name remains undecided. The eventual name should communicate collaborative, question-driven, full-stack continuity without claiming autonomous development. Repository, package, marketplace, domain, and trademark availability must be checked at the time the user chooses a name.

Renaming is not part of this UI workspace implementation and must not block it.

## 14. Security and failure behavior

The design preserves the existing import limits and adds workspace-aware postconditions.

It must reject or safely stop on:

- archive traversal, links, executable payloads, likely secrets, or excessive expansion;
- credentials embedded in Git remotes;
- promotion outside the selected UI workspace;
- overwrite of existing canonical files without an explicit reviewed selection;
- dirty or diverged repositories when an exact baseline is required;
- a UI commit unavailable from its recorded remote;
- root pointer and child HEAD mismatch;
- missing license or source identity where redistribution is requested;
- a frontend integration claiming a UI baseline it did not test;
- stale UI work hidden by a context cache.

Failures leave inspected source material and recoverable state intact. They do not silently delete, commit, push, or reset user work.

## 15. Acceptance criteria

Implementation is complete only when automated tests and dogfood scenarios prove:

1. a new orchestration project can add and register an existing `ui/` submodule safely;
2. an existing single- or multi-repository project can adopt a UI workspace non-destructively;
3. `whole`, `selected`, and `reference-only` source promotion obey safety and provenance rules;
4. imported material can be edited and committed normally in `ui/`;
5. frontend work is bound to an exact clean, remotely available UI baseline;
6. changed UI meaning or pointer state makes dependent frontend work stale or blocked as appropriate;
7. clone recovery reconstructs root, UI, frontend, backend, source, work, and integration state;
8. child workspace recovery locates the root through its bridge;
9. worktrees and semantic claims prevent or order conflicting UI and frontend work;
10. TDD and integration evidence identify the exact UI baseline tested;
11. optional UI Skills are absent without breaking repo-local continuation;
12. macOS and Windows builds and path handling pass;
13. Korean and English README and guides remain behaviorally equivalent;
14. no operation commits, pushes, creates remote repositories, or overwrites user work implicitly.

## 16. Critical evaluation

This feature strengthens the product only if the boundary stays narrow.

Its differentiated value is not that it generates attractive UI. Many tools already do that. Its value is that a generated, imported, or externally approved UI becomes an editable, versioned product baseline whose exact meaning and commit can be coordinated with business rules, backend contracts, multiple repositories, active work, tests, and a release candidate.

The design becomes over-engineered if the product attempts to replace design applications, convert every artifact format, host previews, or prescribe a universal UI schema. Those capabilities remain optional tools or project-specific extensions.

The chosen design therefore adds deterministic identity, safety, continuity, and integration around UI work while keeping creative production flexible.
