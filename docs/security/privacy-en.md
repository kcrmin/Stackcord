# Privacy

The core is local-first and has no telemetry, daemon, central service, or required account. Raw conversation, prompts, source file content, secrets, and unnecessary personal data are not canonical project state.

Repositories store normalized decisions, stable IDs, fingerprints, provider links, claims, and reproducible evidence. Credentials stay in operating-system or provider credential stores and environment variables; configuration records only the environment variable name.

Diagnostic exports omit source content and provider payloads, replace home/project paths with symbolic labels, remove credentials from URLs, redact secret-like values, and include only versions, architecture, stable error codes, redacted state, and operation receipt IDs. Review an export before sharing it because project-specific identifiers may still be sensitive.

External adapters are optional. A provider write requires scoped approval and an idempotency receipt. Removing the Plugin never removes project-owned specs or contracts.
