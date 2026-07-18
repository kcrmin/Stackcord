# Getting started

## Prerequisites

Use Git for collaboration; Git is required when a release candidate must be traceable. A Plugin-capable AI client improves discovery, but the generated repository also includes a standalone Skill and Markdown fallback. Go 1.26 or newer is needed only when building from source.

## Install a verified release bundle

Download the Plugin zip for the current platform together with `checksums.txt`, verify its SHA-256, and unpack it. The bundle contains `.agents/plugins/marketplace.json`, the five Skills, lifecycle hooks, project templates, both bootstrap scripts, and `distribution/platform.json`. That platform record binds the Plugin version to the matching CLI asset and checksum URL.

Ask the AI “Install this verified bundle locally.” It can inspect the platform record and run the matching checksum-first bootstrap. To install the unpacked Plugin through Codex CLI, add its directory as a local marketplace and install the listed Plugin from that marketplace:

```bash
codex plugin marketplace add /absolute/path/to/unpacked/fullstack-orchestrator
codex plugin add fullstack-orchestrator@fullstack-orchestrator
```

The public package name and URL will replace this working name at publication. The bootstrap accepts only HTTPS release URLs, except loopback HTTP used by tests; it verifies the checksum and a `doctor` smoke test before atomically replacing the CLI. Hooks never download or install software.

## Build the CLI

From the product repository:

```bash
cd cli
go test ./...
go build -o ../bin/orchestrator ./cmd/orchestrator
```

Windows PowerShell uses `go build -o ..\bin\orchestrator.exe .\cmd\orchestrator`. Put the resulting binary on `PATH` or tell the AI its absolute path. Run `orchestrator doctor --json` to inspect Git and optional capabilities. This source-build path is for contributors; ordinary users should prefer the verified bundle.

## Install the optional Plugin

For source-tree development, add this repository as a local marketplace and install it from **Plugins** or Codex CLI:

```bash
codex plugin marketplace add /absolute/path/to/fullstack-orchestrator
```

In Codex CLI, open `/plugins` after adding the marketplace. For a GitHub-hosted marketplace use `codex plugin marketplace add owner/repo`. Plugin installation is optional; generated projects retain repo-local behavior.

## Start by talking to the AI

Say “Start a new service with me” in an empty parent directory or “Adopt this existing project without overwriting my files” in an existing repository. The AI inspects the filesystem and Git first, loads the relevant Skill, and asks one material question at a time. It saves normalized checkpoints as discovery continues.

After initialization, use ordinary requests such as “What should I do next?”, “Build this feature”, “Check the contract and database impact”, or “Prepare a production candidate”. You should not need to manage internal IDs or command arguments.

## Verify the first result

Confirm that `README.md`, `AGENTS.md`, `.agents/skills/use-project-harness/`, `.harness/`, `specs/`, `contracts/`, and `docs/` exist. Ask the AI to run a context audit and Git inspection. The audit must use repository files as evidence and must report unknown or stale items instead of inventing answers.

## Next guides

Read [Core concepts](../concepts/en.md), then choose [New project](../guides/new-project-en.md) or [Existing project](../guides/existing-project-en.md). See [Task management and work reservation](../guides/task-management-en.md) before coordinated parallel work. Use [Troubleshooting](../guides/troubleshooting-en.md) when clone, context, Git, or optional-tool state is unclear.
