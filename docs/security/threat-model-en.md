# Threat model

Protected assets include source code, product policies, contracts, Git history, credentials, provider data, imported artifacts, release identities, and user approvals. Trust boundaries include the local machine, untrusted repositories, Git remotes, task providers, dbdiagram, imported archives, CI, package registries, and production targets.

Primary threats are instruction injection from repository files, path/symlink/junction escape, malicious archives and decompression bombs, command/argument injection, secret leakage, credential-bearing remote URLs, hidden Git mutation, stale or fabricated provider state, duplicate external writes, semantic contract conflicts, submodule pointer substitution, unsafe Hooks, dependency compromise, and RC substitution after user validation.

Controls include nearest trusted root discovery, strict schemas and duplicate-key rejection, canonical fingerprints, read-only default diagnosis, A–D approval classes, shell-free allowlisted Git reads, operation journals and idempotency receipts, import quarantine and size limits, environment-only secrets and redaction, exact submodule pins, semantic claims, capability negotiation, immutable RC digests, signed artifacts, SBOM/provenance, and same-RC user verification.

Residual risks must be recorded as owned warnings with rationale. Production publishing always needs exact approval and can be disabled by organization policy.
