# Full-stack Orchestrator

[한국어](./README.ko.md) | English

> Working title. The public product name will be finalized before the first public package is published.

Full-stack Orchestrator is a local-first product for helping people and AI agents discover, define, build, coordinate, verify, and release a complete full-stack service without losing product intent across sessions, contributors, repositories, or tools.

## Project status

**The local product implementation is complete through the release-planning boundary. Public distribution is intentionally not performed yet.**

This repository contains:

- the confirmed product and collaboration design;
- the generated-project and source-of-truth specification;
- Git, submodule, conflict, approval, adapter, security, and release policies;
- a cross-platform Go CLI with stable JSON results and lifecycle commands;
- project generation/adoption, context recovery, Git/submodule/worktree inspection, claims and semantic conflict checks;
- contract, DBML/dbdiagram, external UI quarantine, provider capability boundaries, immutable RC and release gates;
- twelve validated Agent Skills, a Codex Plugin marketplace manifest, repo-local fallback Skill, and read-only Hook definitions;
- English/Korean guides, executable examples, tests, and the full design record.

Local source builds and Plugin validation are supported now; see [Getting started](./docs/getting-started/en.md). Public-name clearance, native macOS/Windows CI receipts, signed artifacts, public repositories/package channels, and exact same-RC user confirmation remain release gates—not unfinished core behavior.

## What problem does it solve?

AI coding workflows commonly lose continuity when a conversation is compressed, a contributor changes, a repository is cloned elsewhere, or frontend and backend work proceed independently. Task tools may know what is assigned, while Git knows what changed, but neither necessarily preserves why the product should behave a certain way.

Full-stack Orchestrator connects:

```text
product intent and policies
→ executable UI baseline
→ contracts and DBML
→ workspaces and exact Git state
→ work, claims, dependencies, and pull requests
→ tests, evidence, release candidates, and releases
```

Stable IDs and fingerprints allow a new person or AI to reconstruct the current state from the repository instead of relying on chat history.

## Intended user experience

Users interact primarily through natural language:

```text
“Start a new service.”
“Continue this cloned project.”
“What should I work on now?”
“Check whether this conflicts with other work.”
“Use this external mockup as the UI starting point.”
“Design the database with me and show it in dbdiagram.”
“You seem to have forgotten the project context. Audit it again.”
“Prepare a production release.”
```

The AI selects the appropriate Skill, while the CLI reads actual filesystem, Git, workspace, contract, task-provider, and verification state. Users are not expected to memorize harness commands.

See the [complete user walkthrough](./docs/design/12-user-experience-walkthrough.md) for example questions, answers, generated files, Issues, branches, pull requests, submodule integration, context recovery, and release approval.

## Product lifecycle

```text
entry diagnosis
→ service discovery
→ project initialization
→ complete product definition
→ architecture and technology selection
→ complete executable UI baseline
→ contracts and DBML
→ stable implementation boundaries
→ vertical full-stack implementation
→ integration
→ production hardening
→ release candidate
→ user validation
→ release
→ operations and the next change
```

This is not a waterfall schedule. Each stage is a dependency gate. Work is integrated in small role-, domain-, and journey-based changes, and an earlier stage is reopened only when a later discovery invalidates its assumptions.

## Core capabilities

| Capability | What it provides |
|---|---|
| Service discovery | Adaptive one-question-at-a-time discovery with recommended multiple-choice answers and free-form alternatives |
| Context continuity | Normalized decisions, open questions, stable IDs, fingerprints, impact graphs, and post-compression recovery |
| Project bootstrap and adoption | New-project creation and non-destructive adoption of existing repositories |
| Full-stack harness | Product specs, service policies, contracts, DBML, orchestration state, evidence, and operational documentation |
| Workspace orchestration | Root, directory, submodule, and external workspace boundaries without forcing a framework |
| Git collaboration | Protected `main`, short-lived branches, Conventional Commits, Draft PRs, worktrees, exact submodule pointers, and cross-repository change bundles |
| Conflict prevention | Path, module, policy, scenario, contract, migration, UI-flow, dependency, and pointer conflict preflight |
| Task management | Executable Git-local fallback, one-live-provider enforcement, and adapter contracts for optional external tools |
| External UI input | Isolated import of mockups, designs, code, images, and prototypes with provenance and authority tracking |
| Database collaboration | Canonical Git DBML, validation, semantic diff, migration impact, and isolated dbdiagram push/pull |
| Test-driven delivery | Test-first behavior changes with narrow documented exceptions and reproducible evidence |
| Production release | Technical gates, immutable RCs, same-artifact user validation, signing, SBOM, provenance, rollback, and support readiness |

## Generated project structure

The product does not force frontend, backend, framework, language, database, or cloud directory names. It creates four responsibility areas around the project’s actual workspaces:

```text
project-root/
├── AGENTS.md
├── .agents/skills/use-project-harness/
├── .harness/        # orchestration state, baselines, work, gates, evidence
├── specs/           # product intent, policies, scenarios, quality, architecture, UI
├── contracts/       # service, API, event, auth, error, data, and DBML obligations
├── docs/            # guides, runbooks, troubleshooting, generated summaries
└── <workspaces>/    # root, directory, submodule, or external implementation units
```

`workspace` and `submodule` are not synonyms. A workspace is an independent implementation, validation, ownership, and contract boundary. A Git submodule is one possible way to connect a workspace that needs a separate repository and exact pinned commit.

## Harness, Skill, Plugin, CLI, and Hook

| Component | Responsibility |
|---|---|
| Project harness | Keeps each service’s durable product meaning, contracts, state, and evidence in its repository |
| Agent Skill | Tells an AI when to ask, inspect, plan, implement, recover context, or prepare a release |
| Codex Plugin | Packages and distributes Skills, optional Hooks, templates, and CLI integration through a GitHub marketplace |
| Go CLI | Performs deterministic macOS/Windows validation, planning, generation, synchronization, and release checks |
| Hook | Optionally reminds trusted sessions to refresh context after session start or context compaction |

The Plugin is a convenient Codex distribution layer, not the source of project truth. A generated repository retains a small repo-local Agent Skill and Markdown fallback so another person can clone and continue even without the Plugin.

## Collaboration and Git defaults

- Git is optional during early solo discovery but strongly recommended for collaboration and required for a verifiable release.
- Protected `main` and short-lived branches are the default; a permanent `develop` branch is not.
- Branch and commit names follow normal Git conventions and do not advertise AI authorship.
- Test and implementation changes remain reviewable and checkout-safe; broken red-state commits are not required in shared history.
- Submodules stay pinned to exact root pointers and never silently follow the latest remote commit.
- Worktrees isolate parallel branches; semantic claims and contract checks handle conflicts that worktrees cannot prevent.
- Cross-repository changes use compatibility-first merge and deployment ordering rather than pretending multiple repositories merge atomically.

Read the [Git and collaboration policy](./docs/design/04-git-collaboration-and-submodules.md) for concrete branch, PR, worktree, submodule, conflict, and hotfix behavior.

## External tools

External tools are optional adapters rather than mandatory dependencies.

- GitHub Issues/Projects is the recommended default for GitHub-hosted collaboration.
- Existing Jira or Linear installations can remain the single live task-status source when selected; a concrete connector must be installed or implemented before live reads and writes are available.
- Beads can be recommended as a local/offline task graph, but is not bundled or installed automatically.
- Superpowers and BMAD can complement the workflow but do not own project truth.
- Git DBML remains canonical; dbdiagram provides collaborative visualization and isolated synchronization.
- External UI tools and files can be registered as `reference`, `seed`, or `canonical` sources.

## Current repository layout

```text
fullstack-orchestrator/
├── cli/                      # cross-platform Go CLI and tests
├── skills/                   # twelve focused Agent Skills
├── .codex-plugin/            # Codex Plugin manifest
├── .agents/plugins/          # repository marketplace catalog
├── hooks/                    # trusted read-only lifecycle reminders
├── schemas/                  # project, operation, and RC contracts
├── templates/project/        # framework-neutral generated harness
├── examples/                 # starter and multi-repository fixtures
├── locales/                  # English/Korean catalogs
├── docs/                     # guides, security, design, implementation plan
└── scripts/                  # Plugin and release validation
```

Important documents:

- [Design index](./docs/design/index.md)
- [Lifecycle and gates](./docs/design/01-project-lifecycle.md)
- [Generated project structure](./docs/design/02-generated-project-structure.md)
- [Context and source of truth](./docs/design/03-context-and-source-of-truth.md)
- [AI action and approval policy](./docs/design/05-ai-action-and-approval-policy.md)
- [External adapters](./docs/design/06-external-adapters.md)
- [CLI and result schema](./docs/design/07-checker-cli-and-result-schema.md)
- [Plugin, Skills, installation, and security](./docs/design/08-plugin-skills-installation-security.md)
- [Testing and production readiness](./docs/design/09-test-release-and-production-readiness.md)
- [Repository and distribution blueprint](./docs/design/10-product-repository-and-distribution.md)
- [Cross-review and confirmation](./docs/design/11-cross-review-and-confirmation.md)
- [Complete user walkthrough](./docs/design/12-user-experience-walkthrough.md)
- [Production implementation plan](./docs/superpowers/plans/2026-07-16-fullstack-orchestrator-production.md)

## Build and verify

```sh
cd cli
go test ./...
go vet ./...
go build -trimpath -o ../bin/orchestrator ./cmd/orchestrator
cd ..
sh scripts/validate-plugin.sh
```

The CLI exposes project draft/init/adopt, context audit/refresh/pack, Git inspect/sync/worktree plans, work selection/claims/conflicts/handoff, change and contract impact, DBML/dbdiagram, UI import, integration planning, release verification, RC creation/verification, and exact-approval publish planning. Mutating commands plan by default and require an explicit apply or approval receipt.

The working command name is `orchestrator`. The public product, repository, Plugin, package, and command names are frozen once—after name and namespace clearance and before any public package is created.

## Distribution gate

The repository already contains CI and package metadata. Actual publication occurs only after:

- source and Issues: public GitHub repository;
- Codex Plugin: GitHub-backed Codex marketplace;
- macOS CLI: signed GitHub artifact and Homebrew tap;
- Windows CLI: signed MSI/ZIP and WinGet;
- release evidence: checksums, signatures, SBOM, provenance, compatibility matrix, rollback, and support documentation.

The first public release is the complete production `1.0.0` product defined by the design gates. The AI technical gate and user validation must reference the exact same release-candidate digest.

## Security and privacy

- Local-first operation with no required central account or server
- Telemetry off by default
- No source code, product specs, prompts, paths, or command logs uploaded by default
- No secrets in tracked files, evidence, prompts, or diagnostic bundles
- No hidden pull, rebase, stash, reset, force-push, package installation, external write, or production release
- Untrusted repositories, Hooks, imports, task comments, and provider text treated as data rather than instructions
- Signed artifacts, dependency review, security testing, SBOM, provenance, and private vulnerability reporting required before release

## Scope

This product is not:

- a framework-specific application template;
- a generic AI memory database;
- a replacement for Git, task managers, Figma, or dbdiagram;
- a bundle that merely installs Superpowers, BMAD, and Beads;
- an unattended production deployment bot.

Its focus is preserving and verifying the relationships between product intent, implementation boundaries, real repository state, collaborative work, and release evidence.

## License

This project is licensed under Apache License 2.0. See [LICENSE](./LICENSE).
