#!/usr/bin/env python3
"""Fail closed unless a workflow is bound to one approved candidate and protected tag."""

import json
import os
import pathlib
import re
import sys


SEMVER_TAG = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")


def verify(root: pathlib.Path, environment: dict[str, str]) -> list[str]:
    errors: list[str] = []
    tag = environment.get("EXPECTED_TAG", "")
    digest = environment.get("EXPECTED_RC_DIGEST", "")
    operation = environment.get("APPROVAL_OPERATION_ID", "")
    ref = environment.get("GITHUB_REF_VALUE", "")
    if not SEMVER_TAG.fullmatch(tag):
        errors.append("release tag must be exact stable semver")
    if not DIGEST.fullmatch(digest):
        errors.append("RC digest must be exact sha256")
    if not operation.startswith("release-approval-"):
        errors.append("class D approval operation receipt is missing")
    if ref != f"refs/tags/{tag}":
        errors.append("workflow ref is not the approved protected tag")
    candidate_path = root / "release" / "approved-rc.json"
    if not candidate_path.is_file():
        errors.append("release/approved-rc.json is absent")
        return errors
    try:
        candidate = json.loads(candidate_path.read_text(encoding="utf-8"))
    except Exception as error:
        errors.append(f"approved candidate is invalid JSON: {error}")
        return errors
    if candidate.get("digest") != digest:
        errors.append("approved candidate digest differs from workflow input")
    if candidate.get("input", {}).get("version") and tag != "v" + candidate["input"]["version"]:
        errors.append("approved candidate version differs from tag")
    if not candidate.get("input", {}).get("user_validation_digest"):
        errors.append("same-RC user validation receipt is absent")
    return errors


def main() -> int:
    errors = verify(pathlib.Path.cwd(), dict(os.environ))
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print("Production publish guard passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
