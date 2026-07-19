import pathlib
import tempfile
import unittest

import generate_packages


ROOT = pathlib.Path(__file__).resolve().parents[3]


class PackageGenerationTest(unittest.TestCase):
    def test_replaces_every_release_artifact_checksum(self):
        checksums = "\n".join([
            "a" * 64 + "  stackcord_1.0.0_Darwin_arm64.tar.gz",
            "b" * 64 + "  stackcord_1.0.0_Darwin_x86_64.tar.gz",
            "c" * 64 + "  stackcord_1.0.0_Windows_arm64.zip",
            "d" * 64 + "  stackcord_1.0.0_Windows_x86_64.zip",
        ])
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            generate_packages.generate(ROOT, output, "1.0.0", checksums)
            files = list(output.rglob("*"))
            self.assertTrue(any(path.suffix == ".rb" for path in files))
            self.assertTrue(any(path.suffix == ".yaml" for path in files))
            text = "\n".join(path.read_text() for path in files if path.is_file())
            self.assertNotIn("{{", text)
            for digest in ("a" * 64, "b" * 64, "c" * 64, "d" * 64):
                self.assertIn(digest, text)


if __name__ == "__main__":
    unittest.main()
