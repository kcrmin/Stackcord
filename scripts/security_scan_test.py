import pathlib
import tempfile
import unittest

import security_scan


class SecurityScanTest(unittest.TestCase):
    def test_detects_credentials_and_allows_environment_variable_names(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "safe.yaml").write_text("secret_environment: DBDIAGRAM_TOKEN\n")
            self.assertEqual([], security_scan.scan(root))
            (root / "leaked.txt").write_text("https://alice:password@github.com/private/repo.git\n")
            self.assertEqual(["credential-url: leaked.txt"], security_scan.scan(root))


if __name__ == "__main__":
    unittest.main()
