import json
import hashlib
import pathlib
import tempfile
import unittest

import verify_publish_guard


class PublishGuardTest(unittest.TestCase):
    def test_requires_same_candidate_tag_digest_and_user_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            validation = b'{"candidate":"validated","schema_version":1}\n'
            (root / "release" / "user-validation-receipt.json").write_bytes(validation)
            validation_digest = "sha256:" + hashlib.sha256(validation).hexdigest()
            candidate = {"schema_version": 1, "input": {"version": "1.0.0", "root_commit": "a" * 40, "workspace_commits": {"root": "a" * 40}, "artifact_digests": {"archive": "sha256:" + "a" * 64}, "schema_versions": {"harness": "1"}, "adapter_versions": {"git": "1"}, "sbom_digest": "sha256:" + "a" * 64, "provenance_digest": "sha256:" + "a" * 64, "signature_digests": {"checksums": "sha256:" + "a" * 64}, "gate_receipts": {"tests": "receipt"}, "docs_fingerprint": "sha256:" + "a" * 64, "user_validation_digest": validation_digest, "gates": {"required_checks_stable": False, "critical_checks_automated": False, "artifacts_signed": False, "migration_rollback_verified": False, "hooks_trusted_read_only": False, "macos_journey_verified": False, "windows_journey_verified": False, "pluginless_continuation": False, "user_validation_matches": False, "warnings": None}}, "digest": ""}
            candidate["digest"] = verify_publish_guard.candidate_digest(candidate)
            digest = candidate["digest"]
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            approval = {"schema_version": 1, "operation_id": "release-approval-01J", "objective": "publish 1.0.0", "repository": "product", "action": "publish_production", "target": digest, "expires_at": "2099-07-17T00:00:00Z", "approved": True, "exact_d_receipt": True}
            (root / "release" / "approval-receipt.json").write_text(json.dumps(approval))
            environment = {"EXPECTED_TAG": "v1.0.0", "EXPECTED_RC_DIGEST": digest, "APPROVAL_OPERATION_ID": "release-approval-01J", "GITHUB_REF_VALUE": "refs/tags/v1.0.0"}
            self.assertEqual([], verify_publish_guard.verify(root, environment))
            environment["EXPECTED_RC_DIGEST"] = "sha256:" + "c" * 64
            self.assertIn("approved candidate digest differs from workflow input", verify_publish_guard.verify(root, environment))

    def test_rejects_missing_or_mismatched_exact_approval_receipt(self):
        digest = "sha256:" + "a" * 64
        environment = {"EXPECTED_TAG": "v1.0.0", "EXPECTED_RC_DIGEST": digest, "APPROVAL_OPERATION_ID": "release-approval-01J", "GITHUB_REF_VALUE": "refs/tags/v1.0.0"}
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            candidate = {"schema_version": 1, "input": {"version": "1.0.0", "user_validation_digest": "sha256:" + "b" * 64}, "digest": digest}
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            errors = verify_publish_guard.verify(root, environment)
            self.assertIn("release/approval-receipt.json is absent", errors)

            approval = {"schema_version": 1, "operation_id": "release-approval-01J", "objective": "publish 1.0.0", "repository": "product", "action": "publish_production", "target": "sha256:" + "c" * 64, "expires_at": "2099-07-17T00:00:00Z", "approved": True, "exact_d_receipt": True}
            (root / "release" / "approval-receipt.json").write_text(json.dumps(approval))
            self.assertIn("approval receipt target differs from approved candidate", verify_publish_guard.verify(root, environment))

    def test_rejects_duplicate_security_fields(self):
        with self.assertRaisesRegex(ValueError, "duplicate JSON key"):
            verify_publish_guard.strict_load('{"target":"one","target":"two"}')


if __name__ == "__main__":
    unittest.main()
