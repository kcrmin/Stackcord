---
schema_version: 1
id: contract.identity.recovery.v1
kind: contract
status: approved
revision: 1
refs: [architecture.workspace.identity-web]
---

Recovery requests are idempotent by request ID. Rate-limit errors are safe to retry only after the supplied timestamp. Providers must not disclose account existence.
