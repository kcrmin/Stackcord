import json
import pathlib
import tempfile
import unittest

import verify_publish_guard


def digest(character):
    return "sha256:" + character * 64


def strict_candidate():
    candidate = {
        "schema_version": 1,
        "input": {
            "profile": "strict-release",
            "version": "1.0.0",
            "root_commit": "a" * 40,
            "workspace_commits": {"workspace.root": "a" * 40},
            "artifact_digests": {"archive": digest("a")},
            "product_fingerprint": digest("b"),
            "docs_fingerprint": digest("c"),
            "contract_fingerprint": digest("d"),
            "tdd_evidence": {"tests": digest("e")},
            "integration_evidence": {"integration": digest("f")},
            "migration_required": False,
            "strict_evidence": {
                "sbom_digest": digest("1"),
                "provenance_digest": digest("2"),
                "signature_digests": {"checksums": digest("3")},
                "supply_chain_receipts": {"security": digest("4")},
            },
        },
        "digest": "",
    }
    candidate["digest"] = verify_publish_guard.candidate_digest(candidate)
    return candidate


class PublishGuardTest(unittest.TestCase):
    def test_requires_same_candidate_tag_digest_and_user_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            candidate = strict_candidate()
            candidate_digest = candidate["digest"]
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            validation = {
                "schema_version": 1,
                "candidate_digest": candidate_digest,
                "confirmed": True,
                "evidence_digest": digest("5"),
                "verified_at": "2099-07-17T00:00:00Z",
            }
            (root / "release" / "user-validation.json").write_text(json.dumps(validation))
            approval = {
                "schema_version": 1,
                "operation_id": "release-approval-01J",
                "objective": "publish 1.0.0",
                "repository": "product",
                "action": "publish_production",
                "target": candidate_digest,
                "expires_at": "2099-07-17T00:00:00Z",
                "approved": True,
                "exact_d_receipt": True,
            }
            (root / "release" / "approval-receipt.json").write_text(json.dumps(approval))
            environment = {
                "EXPECTED_TAG": "v1.0.0",
                "EXPECTED_RC_DIGEST": candidate_digest,
                "APPROVAL_OPERATION_ID": "release-approval-01J",
                "GITHUB_REF_VALUE": "refs/tags/v1.0.0",
            }
            self.assertEqual([], verify_publish_guard.verify(root, environment))

            validation["candidate_digest"] = digest("6")
            (root / "release" / "user-validation.json").write_text(json.dumps(validation))
            self.assertIn(
                "user validation references a different candidate",
                verify_publish_guard.verify(root, environment),
            )

    def test_rejects_core_candidate_for_strict_publication(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            candidate = strict_candidate()
            candidate["input"]["profile"] = "core"
            candidate["digest"] = verify_publish_guard.candidate_digest(candidate)
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            environment = {
                "EXPECTED_TAG": "v1.0.0",
                "EXPECTED_RC_DIGEST": candidate["digest"],
                "APPROVAL_OPERATION_ID": "release-approval-01J",
                "GITHUB_REF_VALUE": "refs/tags/v1.0.0",
            }
            self.assertIn("strict publication requires a strict-release candidate", verify_publish_guard.verify(root, environment))

    def test_rejects_missing_or_mismatched_exact_approval_receipt(self):
        candidate = strict_candidate()
        candidate_digest = candidate["digest"]
        environment = {
            "EXPECTED_TAG": "v1.0.0",
            "EXPECTED_RC_DIGEST": candidate_digest,
            "APPROVAL_OPERATION_ID": "release-approval-01J",
            "GITHUB_REF_VALUE": "refs/tags/v1.0.0",
        }
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            errors = verify_publish_guard.verify(root, environment)
            self.assertIn("release/approval-receipt.json is absent", errors)

            approval = {
                "schema_version": 1,
                "operation_id": "release-approval-01J",
                "objective": "publish 1.0.0",
                "repository": "product",
                "action": "publish_production",
                "target": digest("c"),
                "expires_at": "2099-07-17T00:00:00Z",
                "approved": True,
                "exact_d_receipt": True,
            }
            (root / "release" / "approval-receipt.json").write_text(json.dumps(approval))
            self.assertIn("approval receipt target differs from approved candidate", verify_publish_guard.verify(root, environment))

    def test_rejects_duplicate_security_fields(self):
        with self.assertRaisesRegex(ValueError, "duplicate JSON key"):
            verify_publish_guard.strict_load('{"target":"one","target":"two"}')

    def test_rejects_malformed_commit_and_strict_evidence_identities(self):
        candidate = strict_candidate()
        candidate["input"]["root_commit"] = "main"
        candidate["input"]["strict_evidence"]["sbom_digest"] = "present-but-not-a-digest"
        candidate["digest"] = verify_publish_guard.candidate_digest(candidate)
        environment = {
            "EXPECTED_TAG": "v1.0.0",
            "EXPECTED_RC_DIGEST": candidate["digest"],
            "APPROVAL_OPERATION_ID": "release-approval-01J",
            "GITHUB_REF_VALUE": "refs/tags/v1.0.0",
        }
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))

            errors = verify_publish_guard.verify(root, environment)

            self.assertIn("approved candidate root commit is invalid", errors)
            self.assertIn("strict publication evidence contains an invalid digest", errors)


if __name__ == "__main__":
    unittest.main()
