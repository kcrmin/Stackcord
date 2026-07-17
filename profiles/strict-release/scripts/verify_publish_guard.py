#!/usr/bin/env python3
"""Fail closed unless strict publication is bound to one approved candidate and tag."""

import hashlib
import json
import os
import pathlib
import re
import sys
from datetime import datetime, timezone
from typing import Optional


SEMVER_TAG = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")
GIT_OBJECT_ID = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
OPERATION_ID = re.compile(r"^release-approval-[A-Za-z0-9._-]+$")
INPUT_FIELDS = (
    "profile",
    "version",
    "root_commit",
    "workspace_commits",
    "artifact_digests",
    "product_fingerprint",
    "docs_fingerprint",
    "contract_fingerprint",
    "tdd_evidence",
    "integration_evidence",
    "migration_required",
    "migration_evidence",
    "rollback_evidence",
    "strict_evidence",
)
MAP_FIELDS = {"workspace_commits", "artifact_digests", "tdd_evidence", "integration_evidence"}
STRICT_FIELDS = ("sbom_digest", "provenance_digest", "signature_digests", "supply_chain_receipts")


def valid_identity_map(value: object, pattern: re.Pattern) -> bool:
    return isinstance(value, dict) and bool(value) and all(
        isinstance(key, str) and bool(key) and isinstance(item, str) and pattern.fullmatch(item)
        for key, item in value.items()
    )


def candidate_digest(candidate: dict) -> str:
    source = candidate.get("input", {})
    ordered_input = {}
    for field in INPUT_FIELDS:
        if field not in source:
            continue
        value = source[field]
        if field in MAP_FIELDS and isinstance(value, dict):
            value = {key: value[key] for key in sorted(value)}
        if field == "strict_evidence" and isinstance(value, dict):
            strict = {}
            for strict_field in STRICT_FIELDS:
                if strict_field not in value:
                    continue
                strict_value = value[strict_field]
                if isinstance(strict_value, dict):
                    strict_value = {key: strict_value[key] for key in sorted(strict_value)}
                strict[strict_field] = strict_value
            value = strict
        ordered_input[field] = value
    manifest = {"schema_version": candidate.get("schema_version"), "input": ordered_input, "digest": ""}
    payload = json.dumps(manifest, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    return "sha256:" + hashlib.sha256(payload).hexdigest()


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
    candidate_input = candidate.get("input", {})
    if candidate.get("digest") != digest:
        errors.append("approved candidate digest differs from workflow input")
    if candidate.get("schema_version") != 1 or candidate.get("digest") != candidate_digest(candidate):
        errors.append("approved candidate manifest digest does not match its contents")
    if candidate_input.get("profile") != "strict-release":
        errors.append("strict publication requires a strict-release candidate")
    if not GIT_OBJECT_ID.fullmatch(str(candidate_input.get("root_commit", ""))):
        errors.append("approved candidate root commit is invalid")
    if not valid_identity_map(candidate_input.get("workspace_commits"), GIT_OBJECT_ID):
        errors.append("approved candidate workspace commits are invalid")
    for field in ("artifact_digests", "tdd_evidence", "integration_evidence"):
        if not valid_identity_map(candidate_input.get(field), DIGEST):
            errors.append(f"approved candidate {field} contains an invalid digest")
    for field in ("product_fingerprint", "docs_fingerprint", "contract_fingerprint"):
        if not DIGEST.fullmatch(str(candidate_input.get(field, ""))):
            errors.append(f"approved candidate {field} is invalid")
    if candidate_input.get("migration_required") is True:
        for field in ("migration_evidence", "rollback_evidence"):
            if not DIGEST.fullmatch(str(candidate_input.get(field, ""))):
                errors.append(f"approved candidate {field} is invalid")
    strict_evidence = candidate_input.get("strict_evidence", {})
    if not isinstance(strict_evidence, dict) or any(field not in strict_evidence for field in STRICT_FIELDS):
        errors.append("strict publication evidence is incomplete")
    elif (
        not DIGEST.fullmatch(str(strict_evidence.get("sbom_digest", "")))
        or not DIGEST.fullmatch(str(strict_evidence.get("provenance_digest", "")))
        or not valid_identity_map(strict_evidence.get("signature_digests"), DIGEST)
        or not valid_identity_map(strict_evidence.get("supply_chain_receipts"), DIGEST)
    ):
        errors.append("strict publication evidence contains an invalid digest")
    if candidate_input.get("version") and tag != "v" + candidate_input["version"]:
        errors.append("approved candidate version differs from tag")

    validation_path = root / "release" / "user-validation.json"
    if not validation_path.is_file():
        errors.append("release/user-validation.json is absent")
    else:
        try:
            validation = strict_load(validation_path.read_text(encoding="utf-8"))
        except Exception as error:
            errors.append(f"user validation is invalid JSON: {error}")
        else:
            if validation.get("candidate_digest") != digest:
                errors.append("user validation references a different candidate")
            if validation.get("schema_version") != 1 or validation.get("confirmed") is not True:
                errors.append("user validation is not confirmed")
            if not DIGEST.fullmatch(str(validation.get("evidence_digest", ""))):
                errors.append("user validation evidence digest is invalid")
            if parse_time(validation.get("verified_at", "")) is None:
                errors.append("user validation timestamp is invalid")

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
        "objective": "publish " + candidate_input.get("version", ""),
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
    print("Strict production publish guard passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
