import unittest

import validate_docs


class DocumentationValidatorTest(unittest.TestCase):
    def test_public_contract_requires_reader_focused_stackcord_flow(self):
        old_contract = " ".join((
            *validate_docs.SKILL_NAMES,
            "feature/account-recovery feat(account): AI",
            ".agents/skills/use-project-harness/ .harness/work/provider.yaml",
            "contracts/registry.yaml .harness/local/context/",
            "```mermaid ui/ frontend/ strict ui-workspace",
        ))
        documents = {"README.md": old_contract, "README.ko.md": old_contract}

        errors = validate_docs.public_contract_errors(documents)

        self.assertTrue(any("Stackcord positioning" in error for error in errors))
        self.assertTrue(any("recommended choice conversation" in error for error in errors))
        self.assertTrue(any("external tool recommendation" in error for error in errors))
        self.assertTrue(any("natural-language installation" in error for error in errors))

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

    def test_readmes_keep_maintainer_inventories_in_detailed_guides(self):
        root = validate_docs.ROOT
        english = (root / "README.md").read_text(encoding="utf-8")
        korean = (root / "README.ko.md").read_text(encoding="utf-8")

        self.assertNotIn("## Git and submodule collaboration", english)
        self.assertNotIn("## What does it actually verify?", english)
        self.assertNotIn("## Git·submodule 협업 구조", korean)
        self.assertNotIn("## 무엇을 실제로 검증하나요?", korean)
        self.assertIn("I think we also need a reservation service.", english)
        self.assertIn("예약 서비스도 필요할 것 같아.", korean)
        self.assertIn("`specs/` answers **what the product does and why**", english)
        self.assertIn("`contracts/` defines **what every implementation must obey**", english)
        self.assertIn("`specs/`는 **제품이 무엇을 왜 하는지**", korean)
        self.assertIn("`contracts/`는 **각 구현이 반드시 지켜야 할 의무**", korean)
        self.assertIn("administrator approval", english)
        self.assertIn("관리자 승인", korean)

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

    def test_readme_and_guides_explain_product_authority_governance(self):
        root = validate_docs.ROOT
        english = (root / "docs/guides/governance-en.md").read_text(encoding="utf-8")
        korean = (root / "docs/guides/governance-ko.md").read_text(encoding="utf-8")
        readme = (root / "README.md").read_text(encoding="utf-8")
        readme_ko = (root / "README.ko.md").read_text(encoding="utf-8")
        for token in ("product authority", "proposal", "user.name", "orchestrator governance check"):
            self.assertIn(token, english)
        for token in ("제품 책임자", "제안", "user.name", "orchestrator governance check"):
            self.assertIn(token, korean)
        self.assertIn("People and AI understand the service differently", readme)
        self.assertIn("사람과 AI마다 서비스의 목적·정책·동작을 다르게 이해", readme_ko)
        self.assertIn("governance-en.md", readme)
        self.assertIn("governance-ko.md", readme_ko)


if __name__ == "__main__":
    unittest.main()
