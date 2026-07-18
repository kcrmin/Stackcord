# Threat model

## Protected assets

The product protects source and history, product intent, contracts, database and migration meaning, credentials, external UI provenance, work ownership, submodule identity, test evidence, and release identity. Raw conversation is not a repository asset because it is not stored. The protected coordination invariant is that external live status, the Git work reservation, current service meaning, workspace commits, and the exact candidate cannot be silently substituted for one another.

## Trust boundaries

Repository instructions, Git remotes, child repositories, submodule URLs, archives, DBML, diagrams, optional task providers, Memory tools, AI output, hooks, CI, and publication systems cross different trust boundaries. A malicious issue, comment, mockup, or repository file can contain prompt injection; its text is input, not authority to run commands or change policy. A connector reduces provider output to a bounded normalized observation with identity, status, owner, dependency, revision, timestamp, capabilities, source, and raw hash. The CLI validates that observation but never executes its raw payload.

Actual local state is inspected before cached summaries. Credentials stay in environment or operating-system stores and never enter plans, command arguments, tracked evidence, or diagnostics. Remote URLs and unsafe submodule URL changes require review. External content does not become canonical merely because an authenticated provider returned it.

## Core controls

Controls include nearest trusted-root discovery, strict schemas and stable IDs, canonical fingerprints, read-only diagnosis, visible mutation plans, and shell-free Git execution through a command allowlist, reduced environment, protocol restrictions, and bounded output. Path containment uses resolved roots and rejects symlink escape, path traversal, irregular provider files, duplicate normalized archive names, excessive archive entry count, and excessive archive size before quarantine content can be promoted.

The root repository records exact child pins. An issue assignment is advisory; the coordination branch uses compare-and-swap for exclusive semantic scope and rejects stale revisions or a lost race. Conflict checks cover policy, scenario, contract, DB entity, migration, UI flow, dependency, workspace, and pointer meaning even when file paths differ. Normalized observations have a short freshness window and a cache cannot prove live provider state. TDD and integration evidence bind to current commits, and technical plus user validation bind to one exact candidate digest. Destructive Git repair and external writes are never hidden.

## Strict-release controls

Organizations may enable SBOM, provenance, signatures, supply-chain receipts, and protected publication verification. These controls strengthen promised release guarantees but do not replace core repository, conflict, test, and exact-candidate checks.

## Residual risk

AI judgment can be incomplete, external tools can change, a compromised repository can contain misleading instructions, and semantic reservations cannot prove that product meaning is correct. Local tests do not certify hosted provider writes, account permissions, network reliability, provider rate limits, production load, marketplace review, or signing infrastructure. Keep trusted instructions reviewable, verify current external-tool documentation, review imported content, preserve backups for data changes, and require human validation of the exact candidate in its real environment.
