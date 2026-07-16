#!/usr/bin/env python3
"""Render Homebrew and WinGet manifests from exact GoReleaser checksums."""

import argparse
import pathlib
import re
import shutil


ARTIFACTS = {
    "DARWIN_ARM64_SHA256": "orchestrator_{version}_Darwin_arm64.tar.gz",
    "DARWIN_AMD64_SHA256": "orchestrator_{version}_Darwin_x86_64.tar.gz",
    "WINDOWS_ARM64_SHA256": "orchestrator_{version}_Windows_arm64.zip",
    "WINDOWS_AMD64_SHA256": "orchestrator_{version}_Windows_x86_64.zip",
}


def parse_checksums(text: str) -> dict[str, str]:
    result: dict[str, str] = {}
    for line in text.splitlines():
        match = re.fullmatch(r"([0-9a-fA-F]{64})\s+\*?(.+)", line.strip())
        if match:
            result[match.group(2)] = match.group(1).lower()
    return result


def generate(root: pathlib.Path, output: pathlib.Path, version: str, checksum_text: str) -> None:
    checksums = parse_checksums(checksum_text)
    replacements = {"VERSION": version}
    for token, pattern in ARTIFACTS.items():
        filename = pattern.format(version=version)
        if filename not in checksums:
            raise ValueError(f"missing checksum for {filename}")
        replacements[token] = checksums[filename]

    if output.exists():
        shutil.rmtree(output)
    for source in sorted((root / "packaging").rglob("*")):
        if not source.is_file() or "windows" in source.parts:
            continue
        relative = source.relative_to(root / "packaging")
        target = output / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        text = source.read_text(encoding="utf-8")
        for token, value in replacements.items():
            text = text.replace("{{" + token + "}}", value)
        if "{{" in text:
            raise ValueError(f"unresolved packaging token in {source}")
        with target.open("w", encoding="utf-8", newline="\n") as handle:
            handle.write(text)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--checksums", type=pathlib.Path, required=True)
    args = parser.parse_args()
    generate(args.root.resolve(), args.output.resolve(), args.version, args.checksums.read_text(encoding="utf-8"))


if __name__ == "__main__":
    main()
