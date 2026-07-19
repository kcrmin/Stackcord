#!/usr/bin/env python3
"""Verify complete staged archives, checksums, signatures, SBOMs, and package manifests."""

import hashlib
import pathlib
import sys


TARGETS = ("Darwin_arm64.tar.gz", "Darwin_x86_64.tar.gz", "Windows_arm64.zip", "Windows_x86_64.zip")


def verify(dist: pathlib.Path) -> list[str]:
    errors: list[str] = []
    checksum_path = dist / "checksums.txt"
    bundle_path = dist / "checksums.txt.sigstore.json"
    if not checksum_path.is_file():
        return ["checksums.txt is absent"]
    if not bundle_path.is_file():
        errors.append("Sigstore checksum bundle is absent")
    checksums = {}
    for line in checksum_path.read_text(encoding="utf-8").splitlines():
        parts = line.split(maxsplit=1)
        if len(parts) == 2:
            checksums[parts[1].lstrip("*")] = parts[0].lower()
    archives = [path for path in dist.iterdir() if path.is_file() and any(path.name.endswith(target) for target in TARGETS)]
    for target in TARGETS:
        matching = [path for path in archives if path.name.endswith(target)]
        if len(matching) != 1:
            errors.append(f"expected one {target} archive")
            continue
        path = matching[0]
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        if checksums.get(path.name) != digest:
            errors.append(f"checksum mismatch for {path.name}")
    if not any(path.suffix in {".spdx", ".json"} and "sbom" in path.name.lower() for path in dist.rglob("*")):
        errors.append("archive SBOM is absent")
    manifests = dist / "package-manifests"
    if not (manifests / "homebrew" / "stackcord.rb").is_file() or not list((manifests / "winget").glob("*.yaml")):
        errors.append("Homebrew or WinGet manifests are absent")
    return errors


def main() -> int:
    errors = verify(pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "dist"))
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print("Staged release verification passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
