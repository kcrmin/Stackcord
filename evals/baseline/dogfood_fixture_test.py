import pathlib
import tempfile
import unittest

from dogfood.run import Dogfood


class DogfoodFixtureTest(unittest.TestCase):
    def test_fixture_writes_preserve_lf_bytes_across_clone_normalization(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            driver = Dogfood(root / "unused-cli", root / "workspace", root / "report.json")
            path = root / "contract.md"
            driver.write(path, "approved meaning\nsecond line\n")
            self.assertEqual(b"approved meaning\nsecond line\n", path.read_bytes())
