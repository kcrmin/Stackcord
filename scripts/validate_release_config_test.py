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
    def test_strict_release_assets_are_isolated_from_the_default_product(self):
        strict = ROOT / "profiles" / "strict-release"
        self.assertTrue((strict / "README.md").is_file())
        self.assertTrue((strict / "packaging" / "homebrew" / "orchestrator.rb").is_file())
        self.assertTrue((strict / "packaging" / "windows" / "Product.wxs").is_file())
        self.assertTrue((strict / "scripts" / "generate_packages.py").is_file())
        self.assertTrue((strict / "scripts" / "verify_publish_guard.py").is_file())
        self.assertFalse((ROOT / "packaging").exists())
        self.assertFalse((ROOT / "scripts" / "release").exists())

    def test_release_configuration_is_guarded_and_complete(self):
        validator = load_validator()
        self.assertEqual([], validator.validate(ROOT))

    def test_default_release_is_core_and_strict_supply_chain_is_optional(self):
        goreleaser = (ROOT / ".goreleaser.yaml").read_text(encoding="utf-8")
        release = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        self.assertIn("formats: [binary]", goreleaser)
        for asset in (
            "orchestrator_{{ .Os }}_{{ .Arch }}",
            "checksums.txt",
        ):
            self.assertIn(asset, goreleaser)
        for strict_token in ("sboms:", "signs:", "cosign", "approval_operation_id"):
            self.assertNotIn(strict_token, goreleaser + release)
        self.assertIn("render_plugin_packages.py", release)
        self.assertIn("gh release create", release)


if __name__ == "__main__":
    unittest.main()
