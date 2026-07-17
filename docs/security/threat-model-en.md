# Threat model

## Protected assets

The product protects source and history, product intent, contracts, database and migration meaning, credentials, external UI provenance, work ownership, submodule identity, test evidence, and release identity. Raw conversation is not a protected repository asset because it is not stored.

## Trust boundaries

Repository files, Git remotes, child repositories, archives, diagrams, optional task providers, AI output, hooks, CI, and publication systems cross different trust boundaries. Actual local state is inspected before cached summaries. Credentials stay in environment or operating-system stores and never enter plans, command arguments, tracked evidence, or diagnostics.

## Core controls

Controls include nearest trusted-root discovery, strict parsing and stable IDs, canonical fingerprints, read-only diagnosis and visible plans, shell-free Git inspection, exact submodule pins, semantic conflict claims, import quarantine and limits, TDD and integration evidence, redaction, and candidate digests tied to exact user validation. Destructive Git repair and external writes are never hidden.

## Strict-release controls

Organizations may enable SBOM, provenance, signatures, supply-chain receipts, and protected publication verification. These controls strengthen promised release guarantees but do not replace core repository, conflict, test, and exact-candidate checks.

## Residual risk

AI judgment can be incomplete, external tools can change, a compromised repository can contain misleading instructions, and semantic claims cannot prove that product meaning is correct. Keep trusted instructions reviewable, verify current external-tool documentation, review imported content, preserve backups for data changes, and require human validation of the exact candidate in its real environment.
