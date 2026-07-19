# Support

Before requesting help:

1. Run `stackcord doctor --json`.
2. Run `stackcord context audit --root <project> --json` when a project is involved.
3. Reproduce with the latest supported release and without optional providers when possible.
4. Create a privacy-safe archive with `stackcord doctor --root <project> --export diagnostic.zip --json`, inspect it, and attach it only if appropriate.

Use GitHub Issues for reproducible bugs, documentation gaps, and feature proposals after the public repository exists. Include operating system/architecture, CLI and Plugin versions, stable error codes, expected and actual outcome, and a minimal non-sensitive fixture. Use private vulnerability reporting for security issues.

Community support is best effort. The project provides no emergency production SLA. Keep rollback and operational ownership inside the service using the generated harness.
