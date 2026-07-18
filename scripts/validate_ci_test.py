from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class CIContractTest(unittest.TestCase):
    def test_ci_runs_product_contract_and_dogfood_checks(self):
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
        for token in (
            "validate_docs_test",
            "validate_agent_eval_test",
            "bootstrap_cli_test",
            "render_plugin_packages_test",
            "validate_agent_eval.py",
            "evals/baseline",
            "dogfood/run.sh",
            "goreleaser",
        ):
            self.assertIn(token, ci)

    def test_native_matrix_covers_supported_targets_and_powershell_path(self):
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
        for token in ("macos-14", "macos-15-intel", "windows-2025", "windows-11-arm"):
            self.assertIn(token, ci)
        self.assertIn("go test ./...", ci)
        production = (ROOT / "cli" / "internal" / "command" / "production_e2e_test.go").read_text(encoding="utf-8")
        self.assertIn("dogfood", production)
        self.assertIn("run.ps1", production)

    def test_release_stages_a_draft_only_on_explicit_dispatch(self):
        release = (ROOT / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8")
        self.assertIn("workflow_dispatch:", release)
        self.assertNotIn("push:\n", release)
        self.assertIn("--draft", release)
        self.assertIn("--skip=publish", release)


if __name__ == "__main__":
    unittest.main()
