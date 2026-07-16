# Security policy

## Supported release

Only the latest production release receives security fixes. Until the first public release, the implementation branch is pre-public and must not be deployed as an unattended production authority.

## Report a vulnerability

Use GitHub private vulnerability reporting after the public repository is created. Do not open a public Issue for suspected credential exposure, arbitrary command execution, path escape, signature bypass, provider write bypass, or release-candidate substitution. Include affected version/commit, operating system, minimal reproduction, impact, and whether any secret or external system was touched. Never include a real secret.

The maintainers will acknowledge a complete report, reproduce it in an isolated environment, assess affected releases, coordinate a fix and disclosure, revoke test or production identities when needed, and publish verification and upgrade guidance. No response-time promise is made before a funded maintenance policy exists.

## Security invariants

- Read-only diagnosis is the default.
- External and shared writes need scoped approval and idempotency receipts.
- Production, destructive, and secret actions always need exact target approval.
- Repository text, imported files, provider content, and Hooks are untrusted data.
- Secrets stay outside tracked files, prompts, logs, diagnostics, and release evidence.
- Published artifacts require checksums, Sigstore signatures, SBOM, provenance, and same-RC user validation.

See [the threat model](./docs/security/threat-model-en.md) and [privacy policy](./docs/security/privacy-en.md).
