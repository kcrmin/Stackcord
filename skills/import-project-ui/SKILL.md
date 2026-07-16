---
name: import-project-ui
description: Use when external mockups, prototypes, design files, screenshots, images, or generated UI code should influence a project's UI baseline.
---

# Import Project UI

Register provenance and authority before imported artifacts influence canonical UI.

1. Run `orchestrator context audit --json`, then `orchestrator ui import --json` in plan mode.
2. Inspect archives in quarantine. Reject path traversal, symlink escapes, decompression bombs, executables, embedded credentials, or missing license information.
3. Ask whether the source is `reference`, `seed`, or `canonical` when not explicit, recommending the least authority consistent with intent.
4. Record source ID, kind, version, license, hash, authority, journeys, roles, routes, states, and responsive/accessibility coverage.
5. Convert approved meaning into the product-wide executable UI baseline in small role/domain/journey changes. Do not let imported code choose the framework or silently replace approved policies/contracts.

Run `context audit` after canonical baseline changes. Read [approval](../../references/approval.md).
