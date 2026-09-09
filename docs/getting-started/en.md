# Getting started

## Prerequisites

Use Git for collaboration; Git is required when a release candidate must be traceable. A Plugin-capable AI client improves discovery, but the generated repository also includes a standalone Skill and Markdown fallback. Go 1.26 or newer is needed only when building from source.

## Install from GitHub

```bash
codex plugin marketplace add kcrmin/Stackcord --ref v1.0.0
codex plugin add stackcord@stackcord
```

Start a new Codex conversation after installation so the six Stackcord Skills and lifecycle hooks are loaded from the installed snapshot.

## Install a verified release bundle

Download the Plugin zip for the current platform together with `checksums.txt`, verify its SHA-256, and unpack it. The bundle contains `.agents/plugins/marketplace.json`, the six Skills, lifecycle hooks, project templates, both bootstrap scripts, and `distribution/platform.json`. That platform record binds the Plugin version to the matching CLI asset and checksum URL.

Ask the AI “Install this verified bundle locally.” It can inspect the platform record and run the matching checksum-first bootstrap. To install the unpacked Plugin through Codex CLI, add its directory as a local marketplace and install the listed Plugin from that marketplace:

```bash
codex plugin marketplace add /absolute/path/to/unpacked/stackcord
codex plugin add stackcord@stackcord
```

The bootstrap accepts only HTTPS release URLs, except loopback HTTP used by tests; it verifies the checksum and a `doctor` smoke test before atomically replacing the CLI. Hooks never download or install software.

## Build the CLI

From the product repository:

```bash
cd cli
go test ./...
go build -o ../bin/stackcord ./cmd/stackcord
```

Windows PowerShell uses `go build -o ..\bin\stackcord.exe .\cmd\stackcord`. Put the resulting binary on `PATH` or tell the AI its absolute path. Run `stackcord doctor --json` to inspect Git and optional capabilities. This source-build path is for contributors; ordinary users should prefer the verified bundle.

## Install the optional Plugin

For source-tree development, add this repository as a local marketplace and install it from **Plugins** or Codex CLI:

```bash
codex plugin marketplace add /absolute/path/to/stackcord
```

In Codex CLI, open `/plugins` after adding the marketplace. Plugin installation is optional; generated projects retain repo-local behavior.

## Start by talking to the AI

Say “Start a new service with me” in an empty parent directory or “Adopt this existing project without overwriting my files” in an existing repository. The AI inspects the filesystem and Git first, loads the relevant Skill, and groups independent routine questions with recommended answers. Recommendations remain proposals until submitted; sensitive decisions require explicit answers. It saves normalized checkpoints and reports the current section, remaining sections and approximate question count. Saved decisions are recovered rather than asked again.

After initialization, use ordinary requests such as “What should I do next?”, “Build this feature”, “Check the contract and database impact”, or “Prepare a production candidate”. You should not need to manage internal IDs or command arguments.

## Verify the first result

Confirm that `README.md`, `AGENTS.md`, `.agents/skills/use-project-harness/`, `.harness/`, `specs/`, `contracts/`, and `docs/` exist. Ask the AI to run a context audit and Git inspection. The audit must use repository files as evidence and must report unknown or stale items instead of inventing answers.

## Next guides

Read [Core concepts](../concepts/en.md), then choose [New project](../guides/new-project-en.md) or [Existing project](../guides/existing-project-en.md). See [Task management and work reservation](../guides/task-management-en.md) before coordinated parallel work. Use [Troubleshooting](../guides/troubleshooting-en.md) when clone, context, Git, or optional-tool state is unclear.

When the product needs editable external mockups or an independent UI repository, use [UI workspace and external mockups](../guides/ui-workspace-en.md) to choose the appropriate directory or submodule boundary.

## Recover discovery progress

Ask “Continue the saved questions.” Stackcord shows accepted decisions, the current section, completed/remaining sections and an approximate remaining-question range. Independent routine questions appear together with recommendations; sensitive questions and prerequisites stay explicit. A recommendation is not a submitted answer.

For direct inspection, use `stackcord project discovery --draft <draft-root> --json` before initialization, or `stackcord project discovery --root <project-root> --json` afterward. Omit `--json` for a readable summary; `--locale en` or `--locale ko` overrides the saved language. The command does not change files. Old checkpoints without progress metadata report unknown progress.

The complete optional `discovery` input is shown by `stackcord project checkpoint --help`. Save accepted decisions and remove answered questions together, then update section estimates. After initialization, question/decision text remains in `specs/product/` and section planning lives in `.harness/discovery.yaml`, so a clone can recover without the original draft. Harness readiness means that scope is known and no marked blocking question remains; it neither grants policy approval nor automatically creates the harness.

## Optional control center

Run `stackcord dashboard --root .` and open the printed local URL. The bundled browser UI needs no Node runtime or hosted account. It shows discovery, live GitHub Issues, PR links, your assigned issues and requested reviews, settings, and diagnostics. Stop the command to close the session; it does not notify while closed.

Use `stackcord setup --json` to inspect the local UI preference, or `stackcord setup --ui enable --apply` (also `disable` or `ask`) to record it. Plugins offer this choice during first use; host installation hooks never install software or open windows. Codex and Claude use the same project files and CLI, with separate host manifests and hook adapters. Claude package validation is exercised; executable fixtures are not proof of every host version's session behavior.

Settings require a preview before applying. Shared edits become working-tree proposals; personal language/UI choices remain local. A stale revision from another session is rejected. Interrupted operations leave receipts/locks for inspection rather than silently overwriting newer state. Commit and review shared changes through the normal feature-branch PR flow.
