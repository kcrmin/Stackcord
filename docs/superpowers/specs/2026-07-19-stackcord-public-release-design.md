# Stackcord Public Release Design

## Objective

Publish the completed product as the canonical Codex Plugin repository at `https://github.com/kcrmin/Stackcord` and release `v1.0.0` without retaining public `fullstack-orchestrator` branding.

## Repository decision

The destination repository exists, is public, and has no refs. The current local repository is already independent from Soomgil and contains the product's implementation history, so its history is preserved. The verified release HEAD is pushed as remote `main`; a new Git repository is not initialized and no force push is needed.

## Public identity

- Plugin identifier: `stackcord`
- Marketplace identifier: `stackcord`
- Display name: `Stackcord`
- Publisher: `kcrmin`
- Repository and homepage: `https://github.com/kcrmin/Stackcord`
- Initial public version and tag: `1.0.0` and `v1.0.0`
- Public CLI binary and release asset stem: `stackcord`
- Go module: `github.com/kcrmin/Stackcord/cli`

Internal harness concepts and file formats remain compatible. This is a product-identity migration, not a rewrite of service-discovery or coordination behavior.

## Plugin distribution

The repository root remains the Plugin source. `.agents/plugins/marketplace.json` exposes the root Plugin under the `stackcord` marketplace, and `.codex-plugin/plugin.json` carries the production identity and URLs. Installation is documented as:

```text
codex plugin marketplace add kcrmin/Stackcord --ref v1.0.0
codex plugin add stackcord@stackcord
```

The release contains four platform-specific Plugin zip packages, four separate CLI binaries for macOS Intel/ARM and Windows x64/ARM64, and one SHA-256 checksum manifest. Plugin packages continue to exclude Go source, tests, dogfood, and agent evaluations.

## Release sequence

1. Replace public legacy identifiers and URLs and update tests first where behavior changes.
2. Run Go, Python, schema, documentation, Plugin, security, race, dogfood, cross-build, and package-install verification.
3. Add `https://github.com/kcrmin/Stackcord.git` as `origin` and push the verified HEAD to remote `main` without force.
4. Wait for required GitHub Actions checks on `main` to complete.
5. Create and push the signed-off `v1.0.0` tag at that exact commit.
6. Publish the verified local assets as a non-draft GitHub Release.
7. Install the marketplace and Plugin from the public tag in a clean local Codex configuration and verify the installed snapshot.

## Safety and failure handling

- If the remote gains commits before the push, stop and reconcile instead of overwriting it.
- If deterministic tests or GitHub Actions fail, do not create the public release.
- If a release asset checksum or extracted Plugin validation fails, do not publish the release.
- No real external provider adapters are advertised as built in; the Plugin recommends and connects only selected tools.
- The strict release profile remains optional and is not required for this first public Plugin release.

## Success criteria

- A fresh user can add `kcrmin/Stackcord` as a marketplace and install `stackcord@stackcord`.
- The installed manifest, marketplace, docs, package names, CLI help, and release assets consistently say Stackcord.
- Remote `main`, tag `v1.0.0`, and the GitHub Release point to the same commit.
- Checksums cover every uploaded binary and Plugin package.
