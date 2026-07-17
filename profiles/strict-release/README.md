# Strict release profile

This optional profile contains the product repository's organization-grade publication tooling: SBOM, provenance, Sigstore verification, exact production approval, Homebrew/WinGet/MSI staging, and immutable GitHub Release guards.

Generated projects do not receive these files. Enable `strict-release` only when the team explicitly needs these controls. Core candidate identity, TDD and integration evidence, exact Git/workspace commits, and same-candidate user validation remain mandatory in every profile.

The scripts are fail-closed and intended for the protected workflow in `.github/workflows/release.yml`. They do not publish when run locally.
