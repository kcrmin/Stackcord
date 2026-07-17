# Release readiness

This file records the local production-readiness boundary for the focused full-stack project harness. It is not a publication receipt and does not authorize an external release.

## Product boundary verified

- Five non-overlapping natural-language Skills; repo-local Skill and Markdown fallback remain usable without the Plugin.
- Framework-neutral checkpoint/init/adopt, clone recovery, stable context identity, Git/submodule/worktree diagnosis, semantic conflict claims, contract/DBML/UI coordination, TDD/integration evidence, and exact-candidate verification.
- Core release is the default. Organization supply-chain and publication controls are isolated under `profiles/strict-release/`.
- Unsupported external task adapters and hidden external mutations are not claimed.

## Local verification

The release branch must pass all of the following from a clean checkout:

- Go unit, integration, native-binary E2E, and race tests for every package.
- `go vet`, current `staticcheck`, 15-second context-fingerprint fuzzing, and `govulncheck`.
- Plugin behavior/schema validation, official validation of all five packaged Skills plus repo-local fallbacks, and official Plugin manifest validation.
- English/Korean documentation parity, example context audits, repository secret scan, strict-profile unit tests, workflow linting, and GoReleaser configuration validation.
- CGO-free builds for macOS amd64/arm64 and Windows amd64/arm64.
- A GoReleaser snapshot with archives, SBOMs, and checksums. Local snapshot signing is skipped because a real signing identity belongs to the final publication owner and CI trust boundary.

## E2E journeys

Automated tests use real temporary Git repositories and a compiled native CLI to verify:

1. Repeated normalized discovery, project initialization, commit, clone, Plugin-less context recovery, Git inspection, and next-change planning.
2. Non-destructive adoption that preserves user README content.
3. Real submodule pointer/HEAD/dirty diagnosis, worktree discovery, compatibility-first integration, contract checks, official dbdiagram push/pull planning against an isolated DBML copy, external UI quarantine, and semantic conflict blocking.
4. Core candidate preparation and technical plus user validation against one unchanged digest; changed product identity blocks verification.

## CI boundary

CI defines native macOS ARM64/Intel and Windows x64/ARM64 jobs, a four-target cross-build, fuzz smoke, repository contract checks, Action workflow linting, and a separate vulnerability/CodeQL/dependency-review workflow. Hosted CI results must still be observed after the independent repository is pushed; local cross-compilation is not a substitute for native runner evidence.

## External publication blockers

The following decisions or actions are intentionally not inferred:

- final public product name, binary/package identifiers, and repository owner;
- public repository and marketplace visibility;
- signing identity and organization release environment if strict publication is promised;
- package-channel ownership and whether Homebrew, WinGet, or MSI publication is offered;
- explicit authorization for the irreversible public tag, artifact, marketplace, or deployment action.

Until those are resolved, the repository is locally release-ready but not publicly released.
