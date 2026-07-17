# Getting started

## Prerequisites

Use Git for collaboration and Go 1.24 or newer to build the CLI. Git is required when a release candidate must be traceable. A Plugin-capable AI client improves discovery, but the generated repository also includes a standalone Skill and Markdown fallback.

## Build the CLI

From the product repository:

```bash
cd cli
go test ./...
go build -o ../bin/orchestrator ./cmd/orchestrator
```

Windows PowerShell uses `go build -o ..\bin\orchestrator.exe .\cmd\orchestrator`. Put the resulting binary on `PATH` or tell the AI its absolute path. Run `orchestrator doctor --json` to inspect Git and optional capabilities.

## Install the optional Plugin

Add this repository as a local marketplace, restart the ChatGPT desktop app, and install it from **Plugins**:

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

Read [Core concepts](../concepts/en.md), then choose [New project](../guides/new-project-en.md) or [Existing project](../guides/existing-project-en.md). Use [Troubleshooting](../guides/troubleshooting-en.md) when clone, context, Git, or optional-tool state is unclear.
