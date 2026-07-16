#!/usr/bin/env python3
"""Fail closed unless a workflow is bound to one approved candidate and protected tag."""

import json
import hashlib
import os
import pathlib
import re
import sys
from datetime import datetime, timezone
from typing import Optional


SEMVER_TAG = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")
OPERATION_ID = re.compile(r"^release-approval-[A-Za-z0-9._-]+$")
INPUT_FIELDS = (
    "version", "root_commit", "workspace_commits", "artifact_digests", "schema_versions",
    "adapter_versions", "sbom_digest", "provenance_digest", "signature_digests",
    "gate_receipts", "docs_fingerprint", "user_validation_digest", "gates",
)
GATE_FIELDS = (
    "required_checks_stable", "critical_checks_automated", "artifacts_signed",
    "migration_rollback_verified", "hooks_trusted_read_only", "macos_journey_verified",
    "windows_journey_verified", "pluginless_continuation", "user_validation_matches", "warnings",
)


def candidate_digest(candidate: dict) -> str:
    source = candidate.get("input", {})
    ordered_input = {}
    for field in INPUT_FIELDS:
        value = source.get(field)
        if field in {"workspace_commits", "artifact_digests", "schema_versions", "adapter_versions", "signature_digests", "gate_receipts"} and isinstance(value, dict):
            value = {key: value[key] for key in sorted(value)}
        if field == "gates" and isinstance(value, dict):
            value = {key: value.get(key) for key in GATE_FIELDS}
        ordered_input[field] = value
    manifest = {"schema_version": candidate.get("schema_version"), "input": ordered_input, "digest": ""}
    payload = json.dumps(manifest, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    return "sha256:" + hashlib.sha256(payload).hexdigest()


def file_digest(path: pathlib.Path) -> str:
    return "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()


def parse_time(value: str) -> Optional[datetime]:
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        return parsed if parsed.tzinfo is not None else None
    except (TypeError, ValueError):
        return None


def strict_load(text: str) -> dict:
    def reject_duplicates(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"duplicate JSON key: {key}")
            result[key] = value
        return result

    value = json.loads(text, object_pairs_hook=reject_duplicates)
    if not isinstance(value, dict):
        raise ValueError("release JSON root must be an object")
    return value


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
    if not OPERATION_ID.fullmatch(operation):
        errors.append("class D approval operation receipt is missing")
    if ref != f"refs/tags/{tag}":
        errors.append("workflow ref is not the approved protected tag")
    candidate_path = root / "release" / "approved-rc.json"
    if not candidate_path.is_file():
        errors.append("release/approved-rc.json is absent")
        return errors
    try:
        candidate = strict_load(candidate_path.read_text(encoding="utf-8"))
    except Exception as error:
        errors.append(f"approved candidate is invalid JSON: {error}")
        return errors
    if candidate.get("digest") != digest:
        errors.append("approved candidate digest differs from workflow input")
    if candidate.get("schema_version") != 1 or candidate.get("digest") != candidate_digest(candidate):
        errors.append("approved candidate manifest digest does not match its contents")
    if candidate.get("input", {}).get("version") and tag != "v" + candidate["input"]["version"]:
        errors.append("approved candidate version differs from tag")
    validation_digest = candidate.get("input", {}).get("user_validation_digest")
    validation_path = root / "release" / "user-validation-receipt.json"
    if not validation_digest:
        errors.append("same-RC user validation receipt is absent")
    elif not validation_path.is_file():
        errors.append("release/user-validation-receipt.json is absent")
    elif file_digest(validation_path) != validation_digest:
        errors.append("same-RC user validation receipt digest differs from approved candidate")

    approval_path = root / "release" / "approval-receipt.json"
    if not approval_path.is_file():
        errors.append("release/approval-receipt.json is absent")
        return errors
    try:
        approval = strict_load(approval_path.read_text(encoding="utf-8"))
    except Exception as error:
        errors.append(f"approval receipt is invalid JSON: {error}")
        return errors
    expected = {
        "schema_version": 1,
        "operation_id": operation,
        "objective": "publish " + candidate.get("input", {}).get("version", ""),
        "repository": "product",
        "action": "publish_production",
        "target": digest,
        "approved": True,
        "exact_d_receipt": True,
    }
    for field, value in expected.items():
        if approval.get(field) != value:
            if field == "target":
                errors.append("approval receipt target differs from approved candidate")
            else:
                errors.append(f"approval receipt {field} is invalid")
    expires_at = parse_time(approval.get("expires_at", ""))
    if expires_at is None or expires_at <= datetime.now(timezone.utc):
        errors.append("approval receipt is expired or has an invalid expiry")
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
