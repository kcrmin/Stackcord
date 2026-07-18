# New project

## Begin with discovery

Tell the AI what service you want to create. It first diagnoses the directory and available tools, then asks one question at a time only when the answer changes product behavior, architecture, risk, or scope. Questions normally offer two or three mutually exclusive options, the recommended option first, and free-form input.

After every meaningful answer, the AI updates a normalized checkpoint containing current product facts—not a chat transcript. It also raises overlooked privacy, security, accessibility, failure, operations, observability, retention, and abuse cases when they matter.

## Delay technology commitment

Describe capabilities, quality targets, team constraints, deployment environment, data sensitivity, scale, and operational ownership before choosing frameworks or infrastructure. Record technology needs separately from technology choices. When a choice becomes necessary, compare viable candidates and verify their current official maintenance, security, and release status.

## Establish coverage, then slice

Define the service's roles, journeys, policies, failure outcomes, and UI states across the whole product. External mockups may be imported as reference, seed, or canonical input. This baseline is not a frozen waterfall specification: divide it into small role/domain/journey changes and integrate continuously as learning changes the baseline.

## Initialize the harness

When the service has enough identity to create a durable root, ask the AI to initialize the project. It previews the exact files and then creates the minimal framework-neutral harness. Initialize Git early for collaboration. Add child repositories as submodules as soon as an independent workspace is justified, not merely because frontend and backend names exist.

## Implement with shared boundaries

Before parallel work, define interfaces, contracts, DBML, and failure behavior that multiple changes depend on. Reserve shared semantic scope when coordination is needed, create a conventional feature branch or isolated worktree, write the failing test, implement the smallest behavior, and integrate frequently. A small private edit does not require a ticket or reservation. Revise product checkpoints when implementation reveals a real product decision.
