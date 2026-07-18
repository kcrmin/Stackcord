import unittest

import validate_docs


class DocumentationValidatorTest(unittest.TestCase):
    def test_repository_keeps_only_current_design_records(self):
        root = validate_docs.ROOT

        self.assertEqual([], list((root / "docs/superpowers/plans").glob("*.md")))
        self.assertFalse(
            (root / "docs/superpowers/specs/2026-07-17-focused-product-design.md").exists()
        )
        self.assertFalse((root / "compatibility.json").exists())
        self.assertFalse((root / "testdata/releases/valid-input.json").exists())

        agents = (root / "AGENTS.md").read_text(encoding="utf-8")
        self.assertNotIn("docs/superpowers/plans/", agents)

        design_index = (root / "docs/design/index.md").read_text(encoding="utf-8")
        self.assertIn("2026-07-18-service-continuity-harness-design.md", design_index)
        self.assertIn("2026-07-18-ui-baseline-submodule-design.md", design_index)

    def test_extracts_documented_cli_paths_without_arguments(self):
        text = """
Run `orchestrator status --json`, then:

```sh
orchestrator work define --root . --input /tmp/work.json --apply
orchestrator release verify --root . --json
```
"""

        self.assertEqual(
            {("status",), ("work", "define"), ("release", "verify")},
            validate_docs.extract_orchestrator_commands(text),
        )

    def test_public_contract_reports_missing_service_continuity_explanations(self):
        errors = validate_docs.public_contract_errors({
            "README.md": "start-project continue-project",
            "README.ko.md": "start-project continue-project",
            "docs/concepts/en.md": "Memory",
            "docs/concepts/ko.md": "Memory",
            "docs/guides/task-management-en.md": "Git-local",
            "docs/guides/task-management-ko.md": "Git-local",
        })

        self.assertTrue(any("five Skills" in error for error in errors))
        self.assertTrue(any("provider truth" in error for error in errors))
        self.assertTrue(any("AI-free Git conventions" in error for error in errors))
        self.assertTrue(any("generated context location" in error for error in errors))

    def test_safety_contract_reports_missing_external_and_archive_boundaries(self):
        errors = validate_docs.safety_contract_errors({
            "docs/security/threat-model-en.md": "prompt injection",
            "docs/security/threat-model-ko.md": "prompt injection",
            "docs/security/privacy-en.md": "raw provider payload",
            "docs/security/privacy-ko.md": "provider 원본 payload",
            "docs/guides/troubleshooting-en.md": "use Git-local",
            "docs/guides/troubleshooting-ko.md": "Git-local 사용",
        })

        self.assertTrue(any("threat boundaries" in error for error in errors))
        self.assertTrue(any("provider outage" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
