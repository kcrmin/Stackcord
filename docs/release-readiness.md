# Release readiness

This document defines the evidence boundary for the focused full-stack continuity harness. It records local release preparation; it is not a publication receipt and does not authorize a public tag, marketplace entry, hosted-provider write, or package-channel release.

## Product boundary

The product has five non-overlapping natural-language Skills and one generated repo-local Skill. Conversation, product judgment, proactive questions, and optional tool selection stay flexible in the Skills. The Go CLI owns only observable state, safe writes, stable identity, semantic collision, reservation, evidence, and exact release-candidate checks.

Normal use is proportional. A small private documentation change needs no issue, reservation, worktree, or invented TDD test. Shared, interruptible, cross-repository, policy, contract, database, migration, UI-flow, dependency, or pointer work uses a selected task source plus the Git compare-and-swap semantic reservation.

Core release is the default. SBOM, provenance, signature, immutable publication, organization approvals, and package-channel staging remain optional under `profiles/strict-release/`; core snapshots do not claim to generate them.

## Observed local evidence

The latest local release audit must be refreshed after any product change. The 2026-07-18 editable-UI audit observed:

- **179** Go unit, integration, and native-binary E2E test functions passed with cache disabled; the complete suite and complete race suite both exited successfully;
- the representative editable-UI E2E created an orchestration repository, UI submodule, and frontend workspace; promoted and committed an external mockup; bound its exact baseline; then proved that a later UI revision makes prior frontend work stale;
- multi-repository dogfood passed **23/23** assertions using actual root/frontend/backend repositories, submodules, bare remotes, Git reservations, TDD evidence, pointer integration, clean-clone recovery, contract conflicts, provider reconciliation, and exact RC checks (**54** CLI calls and **124** Git calls);
- **52** Python validator tests passed, followed by Plugin, generated repo-local Skill, Markdown fallback, Hook, English/Korean documentation (**12** pairs), agent-evaluation schema, and high-confidence secret/security validation;
- CGO-free macOS amd64/arm64 and Windows amd64/arm64 builds passed. The resulting SHA-256 values were `8a41c35841ef997f59407442d7492152b407c40f1733286e771987bb83726518`, `559e289c2d0f7978e718722169632206c7e03e5386fd5e6f582a01adf4098fa8`, `58ebe5351dadcbd511cc802bc5fa1b6e72f296d4171445dc767712c46a39c38f`, and `bd52997acca6a9b78ba4ee4b868c9cdff9db35438aadabc69f139e2bbac9601e` respectively;
- the pre-existing isolated Codex behavior audit recorded **9/9** once as initial quality evidence. It is historical, ignored local evidence rather than a current CI or release result. CI never invokes the model; a Skill behavior change selects one related scenario, while all nine require the explicit `--all --allow-external-research` opt-in.

Local transcripts and generated diagnostic state stay under `.harness/local/` and are ignored. The checked-in scenario, rubric, dogfood runner, baseline, and test code make the evidence reproducible without publishing private transcripts.

## Production acceptance audit

| # | Requirement | Current evidence and boundary |
|---:|---|---|
| 1 | Discovery, repeated checkpoint, neutral init | `TestCheckpointRevisesOneDiscoveryWithoutRawConversation`, `TestDiscoveryCheckpointExampleIsAValidStartingSnapshot`, focused E2E, and the real `new-project-discovery` drill. The initial request is persisted before the next question; framework selection remains open. |
| 2 | Non-destructive adoption | `TestAdoptExistingProjectPreservesCustomFiles`, tooling-conflict and legacy-upgrade tests, native-binary E2E. Existing README, AGENTS content, ignore rules, and unrelated files are preserved. |
| 3 | Root plus UI/frontend/backend repositories | The editable-UI E2E creates and binds an actual UI submodule and frontend consumer. Dogfood creates actual root, frontend, and backend repositories, bare remotes, recursive submodules, bridges, and exact gitlinks. |
| 4 | Clone, new session, compaction, other client, Plugin-less recovery | Clean-clone, SessionStart/PostCompact Hook, context-audit, generated Skill, and Markdown fallback are tested. Codex is the only Plugin/Hook surface certified; another AI client receives Markdown fallback only and is not claimed as natively verified. |
| 5 | Dirty/ahead/behind/diverged/detached/missing/local-only/pointer safety | `gitx`, continuity, sync, worktree, focused E2E, dogfood, and real local-only drill cover the states and block unsafe mutation. |
| 6 | Work definition and lifecycle | Definition, provider mapping, transition, evidence, scope-fingerprint, parent/child completion, reservation race, and handoff tests plus dogfood cover the lifecycle. Handoff is used for ownership transfer; ordinary session recovery is not mislabeled as handoff. |
| 7 | Git-local and external task providers | Git-local compare-and-swap is exercised against actual remotes. Fresh normalized external assignment, re-read, revision drift, dependency drift, lifecycle sync, clone recovery, and integration identity are exercised without fabricating a SaaS. One historical `current-tool-selection` drill observed 14 web-search events; it is not automated and must run only when a real tool decision needs current official evidence. Real GitHub/Jira/Beads hosted read/write remains an external verification item. |
| 8 | Product authority and semantic collision | Deterministic governance tests reject display-name spoofing, unauthorized or duplicate approvers, stale commits, stale fingerprints, cached observations, and self-escalation. Integration and core release block unapproved protected meaning. Policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and root-pointer overlap remain covered. Real hosted-provider branch protection and account review are external verification items. |
| 9 | Contract-first provider/consumer integration | Contract registry and impact tests plus dogfood prove approved shared behavior, backend provider red/green evidence, frontend consumer/mock red/green evidence, merge ordering, and exact root-pointer integration. |
| 10 | DBML and dbdiagram | Git DBML remains canonical. Semantic DBML diff, isolated official CLI plan, stale proposal rejection, impact evidence, reconciliation, symlink defense, and migration scope/order conflicts are tested. |
| 11 | External UI and editable baseline | Reference-only, selected-file, and whole-source promotion are supported. Accepted files become ordinary editable workspace files; overwrite, traversal, symlink, stale quarantine, authority escalation, and source-fingerprint mismatch are rejected. A clean published UI commit becomes the exact baseline consumed by frontend work, and later baseline changes stale that work. No design provider is silently selected. |
| 12 | Commit-bound evidence | Test, review, integration, migration, rollback, artifact, and user evidence models bind to workspace commits and current meaning; dirty or changed commits stale the evidence. |
| 13 | Exact core and strict RC | Core needs no supply-chain evidence. Strict requires it. Candidate tests and focused E2E reject pointer, contract, data, evidence, manifest, user-validation, and strict-evidence changes. |
| 14 | Cross-platform and security | Local Go tests/race/vet/fuzz/vulnerability/secret scans and four-target builds pass. Native hosted macOS/Windows CI and CodeQL must still be observed after push. |
| 15 | Plugin, Skill, Hook, fallback | Repository validators, Hook tests, Plugin-less E2E, deterministic platform zip rendering, checksum bootstrap, and validation of an unpacked rendered package pass. A clean-host `codex plugin add` drill remains a manual pre-publication check rather than an automated claim. |
| 16 | Bilingual onboarding | Korean/English README, five-minute quickstart, concepts, provider/tool choice, submodule, dbdiagram, external UI, release, security, and troubleshooting guides pass parity and command/path validation. |
| 17 | Dogfood and baseline | `dogfood/report.md` publishes the derived 23/23 and 9/9 results and the 2/9 manual baseline with explicit limitations; it does not invent elapsed-time or team-productivity numbers. |

## CI and distribution boundary

Pull-request CI runs the complete suite on representative macOS ARM64 and Windows x64 hosts, plus repository contracts, dogfood, and four-target cross-builds. It does not repeat the complete suite on macOS Intel and Windows ARM64. Scheduled security checks own race and fuzz; normal release reruns race and renders all four platform artifacts. No ordinary CI or release workflow invokes Codex model evaluation. These workflows remain configuration evidence until they run in the final public repository.

GoReleaser creates unsigned core binaries and checksums for macOS and Windows. Plugin packages embed the matching binary and marketplace metadata. Strict release may stage SBOMs, provenance, signatures, Homebrew, WinGet, or MSI artifacts only after the publication owner supplies the required identity and explicitly enables that profile.

## External verification and publication blockers

These are not inferred or silently performed:

- final public product name, binary/Plugin identifiers, repository owner, and release base URL;
- public GitHub repository and marketplace/publisher visibility;
- a real GitHub and Jira create/assign/status/re-read/drift drill with approved accounts, plus one protected product change reviewed by a configured product authority under real branch protection; Beads verification is needed only if advertised as a tested provider rather than a connector option;
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
