# Getting started

The normal interface is a conversation with an AI: “Start a new service,” “Continue this clone,” or “What should I do next?” The Plugin routes that intent; the CLI supplies deterministic evidence. Direct CLI use is also supported.

## Build the CLI locally

Requirements: Git 2.40+ and Go 1.26+. No framework, database, cloud, Node.js, daemon, account, or telemetry is required.

macOS/Linux shell:

```sh
cd cli
go test ./...
go build -trimpath -o ../bin/orchestrator ./cmd/orchestrator
../bin/orchestrator doctor --json
```

Windows PowerShell:

```powershell
Set-Location cli
go test ./...
go build -trimpath -o ..\bin\orchestrator.exe .\cmd\orchestrator
..\bin\orchestrator.exe doctor --json
```

## Install the Plugin from a GitHub marketplace

After this repository has a public owner/repository identity:

```sh
codex plugin marketplace add OWNER/REPOSITORY --ref main
codex plugin add fullstack-orchestrator@fullstack-orchestrator
```

For a local checkout, add the repository root instead of `OWNER/REPOSITORY`. Restart the ChatGPT desktop app and begin a new task after installation or update.

## First conversation

Say: “Start a new full-stack service.” The AI checkpoints normalized discovery in `.harness-drafts/`, asks one material question at a time, and creates the root only after the service summary and repository name are approved. It does not choose a framework in advance.

In an existing clone, say: “Continue this project.” The AI runs a read-only context audit, reports dirty/diverged/submodule state, active ownership, stale contracts, and one safe next action. It never hides pull, rebase, stash, reset, or pointer movement.

## Verify this repository

```sh
cd cli && go test ./... && go vet ./...
cd .. && sh scripts/validate-plugin.sh
```

Public publishing still requires the identity freeze, native macOS/Windows CI, signed RC artifacts, and exact user confirmation of the same RC digest.
