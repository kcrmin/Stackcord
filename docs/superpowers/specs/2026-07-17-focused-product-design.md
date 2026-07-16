# Focused Full-stack Project Harness Design

## Decision

Build a local-first AI collaboration product, not a general development-orchestration platform.

The AI Skills own conversation, discovery, recommendations, technology research, and product judgment. The Go CLI owns observable state, strict parsing, safe local mutation, Git and submodule inspection, semantic conflict checks, fingerprints, and release-candidate identity. Users interact with the AI; CLI concepts remain implementation details unless a failure needs explanation.

This replaces the broader platform design. The verified implementation at commit `ff02bfb` is the preservation baseline, not the final scope.

## Product outcome

A user can describe a service through a long natural-language conversation, create or non-destructively adopt a framework-neutral project, collaborate through Git/submodules/worktrees with semantic conflict preflight, and let another person or AI clone and continue from repository truth. The product supports contract, DBML/dbdiagram, external UI input, TDD evidence, integration order, and same-candidate release verification without forcing an enterprise release platform on every project.

## Scope classification

### Keep in the default product

- Normalized service discovery with repeatable checkpoints after material answers.
- Framework-neutral project initialization and non-destructive adoption.
- One repo-local Skill plus Markdown fallback for Plugin-less continuation.
- Stable IDs, canonical fingerprints, impact graph, stale and unknown state.
- Actual Git branch, dirty, upstream, ahead/behind, divergence, worktree planning, and fetched remote-claim inspection.
- Exact submodule root pointer, local HEAD, initialization, dirty, unsafe URL, and mismatch inspection.
- Semantic conflict checks across path, policy, scenario, contract, database entity, migration slot, UI flow, dependency major, and root pointer.
- Work claims, handoff when ownership changes, compatibility-first merge order, and TDD evidence.
- Product policy, observable failure behavior, behavioral contracts, canonical Git DBML, isolated dbdiagram pull/diff, and external UI quarantine.
- Core release candidate identity shared by technical and user validation.
- macOS and Windows Go builds, English and Korean user documentation, and context recovery after compaction or repeated questions.
- Internal atomic writes, path/symlink protection, operation journals, strict YAML/JSON decoding, and privacy-safe diagnostics.

### Isolate behind the optional `strict-release` profile

- SBOM, provenance, Sigstore/checksum signatures, organization-owned warnings, protected publication approvals, and supply-chain receipts.
- Immutable public publication workflow and simultaneous marketplace, GitHub Release, Homebrew, WinGet, and MSI staging.
- Detailed A-D approval receipts for production or irreversible organization actions.
- The product repository's own hardened publication workflows and packaging metadata.

The default profile does not display or require these fields. Enabling `strict-release` adds checks; it never weakens core checks.

### Remove or defer

- The unused generic provider registry and placeholder GitHub, generic Git, and dbdiagram provider adapters. Git and dbdiagram core behavior stays in their real modules.
- Claims that Jira, Linear, Beads, or GitHub live integration exists when no concrete connector is installed.
- Seven overlapping Plugin Skills and duplicated policy prose.
- Generated empty lifecycle, baseline, gate, integration, template, and placeholder directories.
- Generated copies of development, TDD, conflict, approval, security, and release policies.
- Duplicate `context pack`, top-level `verify release`, top-level `rc`, and default `release publish` command paths.
- Requiring users to provide operation IDs, approval receipt files, or supply-chain evidence in the normal workflow.
- Future-facing adapter abstractions that have no current executable consumer.

## User-facing Skills

The Plugin exposes exactly five non-overlapping Skills.

1. `start-project`: new-service discovery, repeatable checkpoints, framework-neutral initialization, and existing-project adoption.
2. `continue-project`: clone/resume diagnosis, current-state summary, selected live task source, and next safe work.
3. `plan-project-work`: feature/bug/change planning, technology decision timing, semantic conflict preflight, claim, worktree, and TDD start.
4. `coordinate-project-work`: contracts, failure behavior, DBML/dbdiagram, external UI, handoff, multi-repository/submodule integration, and merge order.
5. `recover-and-release-project`: forgotten/compacted context recovery, production readiness, core RC identity, user validation, and optional strict-release escalation.

Descriptions contain trigger conditions only. Procedures stay in the Skill body. Shared rules live once in three concise references: `workflow.md`, `safety.md`, and `context-recovery.md`.

## Conversation and discovery

The user never manages checkpoint IDs or files directly. The AI asks one material question at a time, normally with two or three mutually exclusive choices, the recommended choice first, plus free-form input. It discovers repository facts itself and actively raises overlooked security, privacy, accessibility, failure, operational, data-lifecycle, and observability concerns when they materially change the product.

After each material answer, the AI sends a complete normalized discovery snapshot to `orchestrator project checkpoint`. The checkpoint contains only structured product meaning:

- summary and current focus;
- roles and journeys;
- capabilities and policies;
- observable scenarios including failure behavior;
- quality and operational requirements;
- UI coverage by role/journey/state;
- technology needs and constraints, without selecting a stack prematurely;
- decisions, assumptions, and open questions.

The CLI validates the snapshot, increments the revision automatically, and atomically replaces the normalized draft files. Raw conversation, tone, prompts, and credentials are never stored. Initialization migrates the approved snapshot into authored `specs/` documents with stable IDs.

## Generated project

A new or adopted project receives only files required for continuation:

```text
project/
├── README.md
├── AGENTS.md
├── .editorconfig
├── .gitattributes
├── .gitignore
├── .agents/skills/use-project-harness/
│   ├── SKILL.md
│   └── references/fallback.md
├── .harness/
│   ├── manifest.yaml
│   ├── entry.md
│   ├── profile.yaml
│   ├── sources.yaml
│   ├── workspaces.yaml
│   ├── state/context-index.json
│   ├── state/impact-graph.json
│   └── work/provider.yaml
├── specs/index.md
├── contracts/registry.yaml
└── docs/index.md
```

No framework directories, empty future directories, provider configs, release candidate, work templates, or copied policy files are generated. Claims, evidence, DBML, contracts, and integration records appear only when real information exists. `profile.yaml` defaults to TDD, Git strongly recommended for collaboration and required for release, one `git-local` task source, and `core` release verification.

## CLI surface

The CLI remains optimized for Skills and CI rather than memorization by users.

- `project checkpoint|init|adopt`
- `context audit|refresh`
- `git inspect|sync-plan|worktree-plan`
- `work next|conflict|start|finish|handoff`
- `change plan`
- `contract check|impact`
- `db diff|diagram`
- `ui import`
- `integrate plan`
- `release prepare|verify`
- `doctor`

All commands return the stable result envelope and domain exit code. Mutation commands plan by default and use generated internal operation identities when applied. Advanced publication is not part of the default CLI surface.

## Collaboration model

Git is optional during early solo discovery, strongly recommended when collaborating, and required for a verifiable release. The default is protected `main`, short-lived conventional branches, Conventional Commits, and isolated worktrees when parallel local changes would overlap.

Claims are intent signals, not locks. A claim records only the active semantic scope and expires. The CLI reads local claims and already-fetched remote feature-branch claims without checkout. It never fetches, pulls, rebases, stashes, resets, force-pushes, or moves a pointer implicitly.

Conflict handling is outcome-based:

- `clear`: proceed independently;
- `coordinate`: assign ownership and merge order;
- `block`: unify shared product meaning, contract, or migration before implementation;
- `unknown`: restore context or external visibility before claiming safety.

Submodule changes merge in compatibility order: additive contract, providers, consumers, UI connection, then exact root pointer. Nested submodules require a separate inspected plan rather than recursive trust.

## External tools

The base product implements Git-local work and does not fabricate external integrations. The AI detects available tools or offers choices only when the project benefits from one, explains the trade-off, and connects only the selected concrete connector. One source owns live task status. GitHub, an existing provider, or a future connector can be added later through a small connector interface, but absent connectors remain explicitly unavailable.

Superpowers, BMAD, and Beads are complementary workflows or task tools. They never own product meaning, contracts, Git state, or release identity.

## Release profiles

### Core

Core release preparation requires:

- coherent canonical context with no blocker;
- clean, exact root and workspace commits;
- artifact digests;
- product/docs/contract fingerprint;
- passing TDD and integration evidence identities;
- migration/rollback evidence only when the project has a migration;
- user-validation receipt bound to the same RC digest.

Changing any included identity creates a new RC.

### Strict release

Strict release additionally requires the organization supply-chain fields and hardened publication workflow. It is enabled explicitly in `.harness/profile.yaml` or by a strict input profile. Product publication metadata remains in this repository under `profiles/strict-release/`; it is not copied into generated projects.

## Failure and recovery

- Invalid or stale context blocks unsafe recommendations but still produces a plain-language summary and one recovery action.
- Missing Plugin or CLI falls back to the repo-local Skill and Markdown procedure with reduced verification clearly stated.
- A compacted or forgetful AI triggers the recovery Skill and performs a fresh read-only audit before mutation.
- Dirty, diverged, detached, unsafe, mismatched, concurrent, symlinked, or unobservable state is never silently reconciled.
- External UI and dbdiagram input remains quarantined until semantic review.

## Test and release acceptance

The final product must pass:

- new project, repeated discovery checkpoint, and existing-project adoption E2E;
- clone/Plugin-less context recovery and post-compaction routing;
- real Git divergence, remote claims, worktrees, submodules, multi-repository merge order, and semantic conflict matrix;
- contract compatibility, DBML semantic changes, dbdiagram isolation, and malicious UI archive cases;
- core and strict RC identity tests;
- macOS and Windows cross-builds and native CI configuration validation;
- full unit, integration, race, fuzz smoke, static analysis, vulnerability, secret, Plugin, documentation parity, and fallback tests.

Actual public naming, public repository/account creation, signing identities, native hosted-CI receipts, and irreversible publication remain external blockers.
