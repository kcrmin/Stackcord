# Full-stack Project Harness

> Working package name. The public product name is intentionally left open until publication.

[한국어](./README.ko.md)

This product helps people define a service with an AI, create or adopt a framework-neutral full-stack repository, and keep developing it safely across contributors, clones, submodules, worktrees, and compressed AI contexts.

The operating rule is simple: **the Skill handles conversation and judgment; the Go CLI verifies actual state, identity, safety, and conflicts.** Users normally speak to the AI instead of memorizing commands.

## What using it feels like

Tell an AI assistant:

- “Start a new service with me.”
- “Continue this project. What should I do next?”
- “Build account recovery.”
- “The database diagram changed. Check why and plan the migration.”
- “Recover the project context and prepare a production candidate.”

The AI loads the matching Skill, reads the repository, runs deterministic checks, asks only questions that materially change the result, and stores normalized product knowledge after important answers. It does not preserve raw conversation or the user's speaking style.

## Capabilities

- Long service discovery with revisioned checkpoints: product summary, roles, journeys, policies, scenarios, quality needs, UI coverage, decisions, assumptions, technology needs, and open questions.
- Framework-neutral project creation and non-destructive adoption of an existing repository.
- Clone recovery using repo-local instructions, stable IDs, fingerprints, context indexes, and stale detection—even without the Plugin.
- Real Git diagnosis: branch, dirty state, upstream, ahead/behind/diverged state, worktrees, and conventional branch planning.
- Exact submodule diagnosis: root-recorded pointer, checked-out HEAD, missing checkout, dirtiness, mismatch, and safe initialization planning.
- Pre-work conflict checks for paths and product meaning: policies, scenarios, contracts, DB entities, migrations, UI flows, dependency majors, and root pointers.
- Time-bounded work claims, explicit handoff when ownership really changes, compatibility-first integration ordering, and TDD evidence.
- Product policy, failure behavior, contract, DBML, migration, external UI mockup, and dbdiagram collaboration flows.
- One production candidate whose technical evidence and user validation must refer to the same digest.
- Optional strict-release profile for organizations that require SBOM, provenance, signatures, and publication receipts.

## Architecture

| Layer | Responsibility |
| --- | --- |
| Five user-facing Skills | Understand natural-language intent, discover product meaning, recommend tools and technology at the right time, and explain results. |
| Cross-platform Go CLI | Read actual repository state and deterministically validate Git, submodules, fingerprints, conflicts, contracts, DBML, UI imports, integration, and release identity. |
| Repository-owned sources | Preserve normalized decisions and current state so another person or AI can recover after clone or context compression. |

The five Skills have non-overlapping entry points:

1. Start or adopt a project.
2. Continue a project and choose the next work.
3. Plan and start a change.
4. Coordinate contracts, DBML, UI, integration, and conflicts.
5. Recover context, harden production, and prepare or verify a release.

## Generated project structure

```text
project/
├── README.md
├── AGENTS.md
├── .agents/skills/use-project-harness/
│   ├── SKILL.md
│   └── references/fallback.md
├── .harness/
│   ├── entry.md
│   ├── manifest.yaml
│   ├── profile.yaml
│   ├── sources.yaml
│   ├── workspaces.yaml
│   ├── state/
│   │   ├── context-index.json
│   │   └── impact-graph.json
│   └── work/provider.yaml
├── specs/index.md
├── contracts/registry.yaml
└── docs/index.md
```

`specs/` owns product meaning and policies. `contracts/` owns cross-component obligations and failure behavior. `.harness/` contains compact machine-readable coordination state. Users rarely edit `.harness/` directly; the AI summarizes it.

## Development flow

The flow is iterative, not waterfall:

1. Diagnose the repository and tools.
2. Discover the service and checkpoint meaningful answers continuously.
3. Initialize or adopt without choosing a framework prematurely.
4. Establish whole-product meaning and UI coverage, then split work by role, domain, and journey.
5. Define shared boundaries, contracts, and DBML before parallel implementation where they reduce ambiguity.
6. Deliver small vertical changes with TDD and continuous integration.
7. Integrate providers before consumers and update the root submodule pointer only after child commits are reviewable.
8. Harden production, prepare one candidate, validate that exact digest technically and with the user, then release and operate.

Technology is selected only after product, quality, team, and operational constraints are known. At selection time, the AI should verify current official maintenance, security, and release information.

## Git, submodules, and worktrees

Git is strongly recommended for collaboration and required for a verifiable release. Branches and commits use normal conventions such as `feature/account-recovery` and `feat(account): add recovery challenge`; they never include AI branding.

Before work begins, the CLI checks local and upstream state and compares the proposed change with active claims. A worktree can isolate simultaneous branches. In a multi-repository project, each child workspace is committed and reviewed in its own repository; the root repository records the exact accepted child commit. A root pointer update is integrated after compatible child work, not on every local commit.

When overlap is detected, the AI explains the conflicting meaning and recommends one of: split ownership, agree on a contract first, sequence provider and consumer changes, merge a shared boundary before parallel work, or deliberately serialize the change. Dirty trees, divergence, detached submodules, and unpublished child commits are reported rather than repaired destructively.

## DBML, dbdiagram, and external UI

Git-tracked DBML is canonical. dbdiagram is an isolated visualization and semantic-diff workspace; a remote diagram change is never silently promoted. The AI asks for the rationale, shows entity-level differences, and connects accepted changes to contracts and migrations.

External mockups are imported into quarantine and registered as `reference`, `seed`, or `canonical`. License, provenance, size, and content are inspected before product files change.

## Core and strict release

Core mode requires repository identity, artifact fingerprints, TDD evidence, integration evidence, applicable migration/rollback evidence, and user confirmation bound to the exact candidate digest. It does not publish anything.

Strict release is opt-in under [`profiles/strict-release`](./profiles/strict-release/README.md). It adds supply-chain and organization gates without burdening ordinary projects. Public account creation, signing identities, irreversible publication, and package-channel ownership remain outside the automatic local workflow.

## Build and test

Go 1.26 or newer is required.

```bash
cd cli
go test ./...
go build -o ../bin/orchestrator ./cmd/orchestrator
```

On Windows PowerShell:

```powershell
cd cli
go test ./...
go build -o ..\bin\orchestrator.exe .\cmd\orchestrator
```

Run `orchestrator doctor --json` to inspect local capabilities. AI assistants use the CLI through the Skills; direct command help remains available with `orchestrator --help`.

## Plugin installation and sharing

The Plugin is optional. A generated repository remains usable through its repo-local Skill and Markdown fallback.

For local development, add this repository as a marketplace source, restart the desktop app, then install it from **Plugins**. Codex CLI users can open `/plugins` after adding the marketplace.

```bash
codex plugin marketplace add /absolute/path/to/fullstack-orchestrator
```

For GitHub distribution, publish the repository and use `codex plugin marketplace add owner/repo`. A team can also keep `.agents/plugins/marketplace.json` in its repository or share an installed local Plugin inside its ChatGPT workspace. See [Getting started](./docs/getting-started/en.md) for the exact local workflow.

## Documentation

- [Getting started](./docs/getting-started/en.md)
- [Core concepts](./docs/concepts/en.md)
- [New project](./docs/guides/new-project-en.md)
- [Existing project](./docs/guides/existing-project-en.md)
- [Submodules and collaboration](./docs/guides/submodules-en.md)
- [DBML and dbdiagram](./docs/guides/dbdiagram-en.md)
- [Release](./docs/guides/release-en.md)
- [Troubleshooting](./docs/guides/troubleshooting-en.md)
- [Focused design](./docs/design/index.md)

## What it is not

It is not a framework generator, an all-purpose project-management platform, or a bundle that treats Superpowers, BMAD, Beads, GitHub Issues, Jira, or Linear as its source of truth. External tools are detected or proposed with trade-offs and connected only after the user selects one live task-status source.

## Before public release

Local implementation and validation can be completed without external decisions. Public publication still requires a final product name and identifiers, a public repository/account, signing ownership if strict artifacts are promised, and explicit authorization for the irreversible release action.
