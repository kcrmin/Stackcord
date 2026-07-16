import importlib.util
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def load_validator():
    path = ROOT / "scripts" / "validate_release_config.py"
    spec = importlib.util.spec_from_file_location("release_validator", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ReleaseConfigurationTest(unittest.TestCase):
    def test_release_configuration_is_guarded_and_complete(self):
        validator = load_validator()
        self.assertEqual([], validator.validate(ROOT))


if __name__ == "__main__":
    unittest.main()
