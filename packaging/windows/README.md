# Windows MSI release boundary

The portable ZIPs and WinGet manifests are built without private credentials. A production MSI is rendered from `Product.wxs` with WiX v4, then Authenticode-signed in a protected Windows release environment. The signing certificate must come from that environment and is never stored in this repository.

The public release gate remains blocked until the signed MSI passes clean install, upgrade, uninstall, PATH refresh, checksum, signature, and Windows ARM64/x64 smoke tests. The repository deliberately does not manufacture a development certificate and call it production signing.
