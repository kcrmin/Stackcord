# Tiered Test Policy Design

## Goal

Keep deterministic product safety while removing repeated platform work and automatic model cost. Agent behavior requirements remain reviewable specifications; actual Codex execution becomes change-selected and explicitly opt-in.

## Classification

| Class | Policy | Checks |
|---|---|---|
| A | Fast and deterministic; run on every pull request | Go tests on representative macOS and Windows, Python validators, schema, documentation parity, secrets, Plugin/package contracts |
| B | Costly but required before a normal release | multi-repository dogfood, four platform binaries, checksums, source and rendered Plugin installation smoke tests |
| C | Run only for relevant changes | race for concurrency changes or release/periodic audit; one selected Codex scenario for a changed Skill |
| D | Explicit manual opt-in only | all nine Codex scenarios, current external-tool research, strict publication checks |
| E | Remove as duplicate or low-value execution | full Go suite on all four native targets, duplicated PR secret scan, development tests inside the Plugin zip |

No automated pull request or normal release workflow may invoke `codex exec` or `run_agent_eval.py`. The nine scenario definitions and rubric remain committed product requirements. Existing local nine-scenario transcripts remain ignored historical evidence and are never described as a current automated result.

## Workflow boundaries

### Development

Contributors run tests for changed Go packages or the relevant Python validator. Normal CLI use and Plugin hooks do not run the development test suite or model evaluations.

### Pull request

- representative macOS ARM runs the complete Go suite, vet, native build, and smoke checks;
- Windows x64 runs the complete Go suite, vet, native build, and smoke checks;
- macOS Intel and Windows ARM64 are not additional full-test jobs;
- Linux runs repository contracts, Plugin package tests, schema/document/secret validation, and workflow/config validation;
- no actual Codex call, dogfood, four-platform release build, or strict publication gate runs.

### Normal release

An explicit dispatch reruns deterministic source checks, race, dogfood, and four CGO-free platform builds. It renders four Plugin packages plus one checksum manifest, installs the source Plugin and one rendered package into isolated temporary Codex homes, and stages a draft release only. No model evaluation runs.

### Skill behavior change

A manual workflow requires exactly one scenario ID and rejects the external-tool-search scenario. The local runner already supports repeated `--scenario`; documentation makes one selection the normal change check. A separate `--all` flag is required to run the complete suite and refuses combination with `--scenario`. The external-tool-search scenario requires its own explicit opt-in flag.

### Strict release

Strict scripts remain an optional extension and are not part of normal pull-request or release gates. Strict tests run only when strict-profile files change or through an explicit strict workflow. Strict test files are excluded from distributed Plugin packages.

## Plugin package boundary

The package contains only the manifest, marketplace metadata, five Skills, Hooks, references, templates, schemas, bootstrap scripts, Plugin validator, README files, LICENSE, and the optional strict runtime files. It excludes Go source and tests, agent evaluations, dogfood, testdata, development documentation, CI, package-rendering code, and every Python `*_test.py` file. The CLI remains a separately downloaded checksum-verified asset.

Source and rendered package installation smoke tests use an isolated `CODEX_HOME` and the real `codex plugin marketplace add` plus `codex plugin add` commands when Codex is available. CI/release must fail rather than silently report installation success if the required release installation check cannot run.

## Evidence and limits

Deterministic validators confirm that workflows contain no automated model invocation and that a single scenario can be selected. Historical model usage is recorded as historical local evidence only. Hosted runner, marketplace, provider, signing, and public release claims remain external until observed.

