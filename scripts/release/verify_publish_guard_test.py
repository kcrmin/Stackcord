import json
import pathlib
import tempfile
import unittest

import verify_publish_guard


class PublishGuardTest(unittest.TestCase):
    def test_requires_same_candidate_tag_digest_and_user_receipt(self):
        digest = "sha256:" + "a" * 64
        environment = {"EXPECTED_TAG": "v1.0.0", "EXPECTED_RC_DIGEST": digest, "APPROVAL_OPERATION_ID": "release-approval-01J", "GITHUB_REF_VALUE": "refs/tags/v1.0.0"}
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "release").mkdir()
            candidate = {"digest": digest, "input": {"version": "1.0.0", "user_validation_digest": "sha256:" + "b" * 64}}
            (root / "release" / "approved-rc.json").write_text(json.dumps(candidate))
            self.assertEqual([], verify_publish_guard.verify(root, environment))
            environment["EXPECTED_RC_DIGEST"] = "sha256:" + "c" * 64
            self.assertIn("approved candidate digest differs from workflow input", verify_publish_guard.verify(root, environment))


if __name__ == "__main__":
    unittest.main()
