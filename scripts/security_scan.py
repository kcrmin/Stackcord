#!/usr/bin/env python3
"""High-confidence repository secret and credential-URL scanner."""

import pathlib
import re
import sys


PATTERNS = {
    "private-key": re.compile(rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    "github-token": re.compile(rb"\b(?:ghp|github_pat)_[A-Za-z0-9_]{30,}\b"),
    "aws-access-key": re.compile(rb"\bAKIA[0-9A-Z]{16}\b"),
    "credential-url": re.compile(rb"https?://[^/\s:@]+:[^@\s/]+@"),
}
SKIP_PARTS = {".git", "dist", ".tools", ".worktrees", "node_modules"}


def scan(root: pathlib.Path) -> list[str]:
    errors: list[str] = []
    for path in root.rglob("*"):
        if not path.is_file() or any(part in SKIP_PARTS for part in path.parts):
            continue
        relative = path.relative_to(root)
        if "testdata" in relative.parts or path.name.endswith(("_test.go", "_test.py")) or relative.as_posix() == "scripts/security_scan.py":
            continue
        if path.name == ".env" or (path.name.startswith(".env.") and path.name != ".env.example"):
            errors.append(f"tracked environment file: {relative}")
            continue
        try:
            data = path.read_bytes()
        except OSError:
            continue
        if b"\x00" in data[:4096]:
            continue
        for name, pattern in PATTERNS.items():
            if pattern.search(data):
                errors.append(f"{name}: {relative}")
    return sorted(errors)


def main() -> int:
    root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = scan(root)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print("High-confidence repository secret scan passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
