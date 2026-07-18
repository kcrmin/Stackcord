from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class CIContractTest(unittest.TestCase):
    def test_ci_runs_product_contract_and_dogfood_checks_without_model_calls(self):
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
        for forbidden in ("run_agent_eval.py", "go test -race", "-fuzz FuzzFingerprint"):
            self.assertNotIn(forbidden, ci)

    def test_pull_request_uses_two_representative_native_full_test_jobs(self):
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
        for token in ("macos-14", "windows-2025"):
            self.assertIn(token, ci)
        for token in ("macos-15-intel", "windows-11-arm"):
            self.assertNotIn(token, ci)
        self.assertIn("go test ./...", ci)

    def test_security_runs_race_and_fuzz_only_on_the_schedule(self):
        security = (ROOT / ".github" / "workflows" / "security.yml").read_text(encoding="utf-8")
        self.assertIn("github.event_name == 'schedule'", security)
        self.assertIn("go test -race ./...", security)
        self.assertIn("-fuzz FuzzFingerprint", security)
        self.assertNotIn("security_scan.py", security)
        self.assertNotIn("go test ./... && go vet ./...", security)

    def test_normal_release_runs_race_and_packaging_without_model(self):
        release = (ROOT / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8")
        for token in (
            "go test -race ./...",
            "render_plugin_packages.py",
        ):
            self.assertIn(token, release)
        self.assertNotIn("run_agent_eval.py", release)

    def test_release_stages_a_draft_only_on_explicit_dispatch(self):
        release = (ROOT / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8")
        self.assertIn("workflow_dispatch:", release)
        self.assertNotIn("push:\n", release)
        self.assertIn("--draft", release)
        self.assertIn("--skip=publish", release)


if __name__ == "__main__":
    unittest.main()
