# Release readiness

This document defines the evidence boundary for the focused full-stack continuity harness. It records local release preparation; it is not a publication receipt and does not authorize a public tag, marketplace entry, hosted-provider write, or package-channel release.

## Product boundary

The product has five non-overlapping natural-language Skills and one generated repo-local Skill. Conversation, product judgment, proactive questions, and optional tool selection stay flexible in the Skills. The Go CLI owns only observable state, safe writes, stable identity, semantic collision, reservation, evidence, and exact release-candidate checks.

Normal use is proportional. A small private documentation change needs no issue, reservation, worktree, or invented TDD test. Shared, interruptible, cross-repository, policy, contract, database, migration, UI-flow, dependency, or pointer work uses a selected task source plus the Git compare-and-swap semantic reservation.

Core release is the default. SBOM, provenance, signature, immutable publication, organization approvals, and package-channel staging remain optional under `profiles/strict-release/`; core snapshots do not claim to generate them.

## Observed local evidence

The latest local release audit must be refreshed after any product change. The 2026-07-18 audit observed:

- actual Codex behavior: **9/9** isolated scenarios passed, including initial checkpoint before A/B/C discovery, clean-clone recovery, context loss, path-disjoint frontend/backend conflict, unavailable Jira, a small private edit, local-only work, mismatched RC validation, and current official-tool comparison without unapproved installation;
- multi-repository dogfood: **23/23** assertions and **9/9** deterministic scenarios passed with actual root, frontend, backend, submodules, bare remotes, concurrent owners, Git reservation, red/green TDD evidence, pointer integration, clean clone, and exact RC checks;
- manual Git plus static docs baseline: **2/9** scenarios had a deterministic native check; this is coverage comparison, not a productivity or speed claim;
- Go unit/integration/native-binary E2E, race, vet, 15-second context fuzzing, and `govulncheck`: passed locally;
- Plugin, Hook, all five packaged Skills, generated repo-local Skill, Markdown fallback, English/Korean parity, secret/security, strict-profile, workflow, release-config, and package validators: passed locally;
- CGO-free macOS amd64/arm64 and Windows amd64/arm64 builds: passed locally;
- clean temporary-clone GoReleaser check and snapshot: passed; the snapshot rendered four native binaries, four Plugin packages, and one checksum manifest;
- source and rendered Plugin packages were both added to a temporary marketplace and installed in a temporary `CODEX_HOME`: passed.

Local transcripts and generated diagnostic state stay under `.harness/local/` and are ignored. The checked-in scenario, rubric, dogfood runner, baseline, and test code make the evidence reproducible without publishing private transcripts.

## Production acceptance audit

| # | Requirement | Current evidence and boundary |
|---:|---|---|
| 1 | Discovery, repeated checkpoint, neutral init | `TestCheckpointRevisesOneDiscoveryWithoutRawConversation`, `TestDiscoveryCheckpointExampleIsAValidStartingSnapshot`, focused E2E, and the real `new-project-discovery` drill. The initial request is persisted before the next question; framework selection remains open. |
| 2 | Non-destructive adoption | `TestAdoptExistingProjectPreservesCustomFiles`, tooling-conflict and legacy-upgrade tests, native-binary E2E. Existing README, AGENTS content, ignore rules, and unrelated files are preserved. |
| 3 | Root plus frontend/backend repositories | Dogfood creates three actual repositories, bare remotes, recursive submodules, bridges, and exact gitlinks. |
| 4 | Clone, new session, compaction, other client, Plugin-less recovery | Clean-clone, SessionStart/PostCompact Hook, context-audit, generated Skill, and Markdown fallback are tested. Codex is the only Plugin/Hook surface certified; another AI client receives Markdown fallback only and is not claimed as natively verified. |
| 5 | Dirty/ahead/behind/diverged/detached/missing/local-only/pointer safety | `gitx`, continuity, sync, worktree, focused E2E, dogfood, and real local-only drill cover the states and block unsafe mutation. |
| 6 | Work definition and lifecycle | Definition, provider mapping, transition, evidence, scope-fingerprint, parent/child completion, reservation race, and handoff tests plus dogfood cover the lifecycle. Handoff is used for ownership transfer; ordinary session recovery is not mislabeled as handoff. |
| 7 | Git-local and external task providers | Git-local compare-and-swap is exercised against actual remotes. Fresh normalized external assignment, re-read, revision drift, dependency drift, lifecycle sync, clone recovery, and integration identity are exercised without fabricating a SaaS. The `current-tool-selection` drill observed 14 real web-search events, compared current official candidates, asked one A/B/C choice, and installed nothing. Real GitHub/Jira/Beads hosted read/write remains an external verification item. |
| 8 | Cross-repository semantic collision | Policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer overlap are tested. Dogfood and the real frontend/backend drill catch a collision with disjoint file paths. |
| 9 | Contract-first provider/consumer integration | Contract registry and impact tests plus dogfood prove approved shared behavior, backend provider red/green evidence, frontend consumer/mock red/green evidence, merge ordering, and exact root-pointer integration. |
| 10 | DBML and dbdiagram | Git DBML remains canonical. Semantic DBML diff, isolated official CLI plan, stale proposal rejection, impact evidence, reconciliation, symlink defense, and migration scope/order conflicts are tested. |
| 11 | External UI | Reference, seed, and canonical authority are represented; quarantine, malicious archive, duplicate path, symlink, authority escalation, stale mapping, and executable-scope integration are tested. No design provider is silently selected. |
| 12 | Commit-bound evidence | Test, review, integration, migration, rollback, artifact, and user evidence models bind to workspace commits and current meaning; dirty or changed commits stale the evidence. |
| 13 | Exact core and strict RC | Core needs no supply-chain evidence. Strict requires it. Candidate tests and focused E2E reject pointer, contract, data, evidence, manifest, user-validation, and strict-evidence changes. |
| 14 | Cross-platform and security | Local Go tests/race/vet/fuzz/vulnerability/secret scans and four-target builds pass. Native hosted macOS/Windows CI and CodeQL must still be observed after push. |
| 15 | Plugin, Skill, Hook, fallback | Repository validators, official Skill validator, real temporary marketplace install, Hook tests, and Plugin-less E2E pass for both source and rendered package. |
| 16 | Bilingual onboarding | Korean/English README, five-minute quickstart, concepts, provider/tool choice, submodule, dbdiagram, external UI, release, security, and troubleshooting guides pass parity and command/path validation. |
| 17 | Dogfood and baseline | `dogfood/report.md` publishes the derived 23/23 and 9/9 results and the 2/9 manual baseline with explicit limitations; it does not invent elapsed-time or team-productivity numbers. |

## CI and distribution boundary

CI defines native macOS ARM64/Intel and Windows x64/ARM64 jobs, race where supported, four-target cross-builds, dogfood, repository contracts, fuzz smoke, vulnerability/CodeQL, dependency review, and workflow linting. Those workflows are configuration evidence until they run in the final public repository.

GoReleaser creates unsigned core binaries and checksums for macOS and Windows. Plugin packages embed the matching binary and marketplace metadata. Strict release may stage SBOMs, provenance, signatures, Homebrew, WinGet, or MSI artifacts only after the publication owner supplies the required identity and explicitly enables that profile.

## External verification and publication blockers

These are not inferred or silently performed:

- final public product name, binary/Plugin identifiers, repository owner, and release base URL;
- public GitHub repository and marketplace/publisher visibility;
- a real GitHub and Jira create/assign/status/re-read/drift drill with approved accounts, plus Beads verification if it is advertised as a tested provider rather than a connector option;
- hosted native macOS/Windows CI, CodeQL, dependency review, and release-workflow results after push;
- signing identity and organization release environment if strict publication is promised;
- package-channel ownership and the decision to publish Homebrew, WinGet, or MSI metadata;
- user beta evidence beyond the reproducible agent and dogfood fixtures;
- explicit authorization for irreversible tag, artifact, marketplace, or package publication.

Until these are resolved, the repository is locally prepared for release but is not publicly released and must not claim hosted-provider or native-runner evidence it has not observed.

## Removed or isolated overdesign

- Unsupported universal Jira/Linear/Beads adapters and speculative provider abstractions were removed; only a selected real connector/CLI may own live status.
- Detailed supply-chain orchestration, SBOM/provenance/signature gates, immutable publication, organization approval receipts, and multi-channel packages are isolated in strict release.
- Operation IDs, receipts, reservation files, and `.harness/` mechanics remain internal and are summarized by the AI.
- Duplicate Skills and copied policy text were collapsed into five public workflows, shared references, schemas, and one generated repo-local entry.
- A task ticket, worktree, reservation, or full lifecycle is not forced onto small private work.
- Superpowers, BMAD, Spec Kit, OpenSpec, Beads, memory tools, GitHub, Jira, dbdiagram, and UI tools remain optional complements selected just in time; none is this product's source of truth by default.
