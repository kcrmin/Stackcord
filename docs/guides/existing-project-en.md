# Existing project adoption

## Inspect before changing

Tell the AI to continue or adopt the repository. It reads the nearest trusted instructions, detects languages and existing tools, inspects Git and submodules, and inventories product documentation, contracts, schemas, tests, CI, deployment, and task tracking. Facts available from files or Git are not asked back to the user.

## Preview a non-destructive plan

Adoption adds only the minimal harness and managed sections. Existing files, settings, source code, branches, task systems, and repository history remain authoritative. If a target file already contains user content, the plan must show whether a delimited managed section can be merged; unsafe collisions block adoption instead of overwriting content.

## Reconstruct product meaning

The AI summarizes what the repository proves, separates facts from assumptions, and asks only material unresolved questions. It assigns stable IDs to existing policies, scenarios, contracts, DB entities, migrations, and UI flows and records their fingerprints. Existing technology stays unless product or operational evidence justifies a change.

## Select one work-status source

Git-local status is the safe default. If the repository already uses GitHub, Jira, Linear, Beads, or another system, the AI may recommend keeping it after confirming a real connector or usable local command exists. Only the selected tool becomes live task-status authority; unsupported adapters are never implied.

## Start the first change

Run context and Git audits, resolve stale or divergent state with the user, select dependency-ready work, and check semantic conflicts before writing code. Preserve existing branch and commit conventions unless they are ambiguous or unsafe. Use TDD and the repository's existing test/build interfaces.
