import unittest
import validate_docs


class DocumentationValidatorTest(unittest.TestCase):
    def test_rejects_broken_links_without_enforcing_marketing_prose(self):
        documents = {
            "README.md": "[core](core/README.md) [runtime](docs/guides/runtime-en.md) [harness](docs/guides/harness-en.md)",
            "README.ko.md": "[core](core/README.md) [runtime](docs/guides/runtime-ko.md) [harness](docs/guides/harness-ko.md)",
        }
        self.assertEqual([], validate_docs.public_contract_errors(documents))
        documents["README.md"] += " [broken](docs/missing-document.md)"
        errors = validate_docs.public_contract_errors(documents)
        self.assertTrue(any("broken local link" in error for error in errors))

    def test_requires_accessible_library_runtime_and_harness_entrypoints(self):
        errors = validate_docs.public_contract_errors({"README.md": "New prose", "README.ko.md": "새 설명"})
        self.assertTrue(any("core/README.md" in error for error in errors))
        self.assertTrue(any("runtime-en.md" in error for error in errors))
        self.assertTrue(any("harness-ko.md" in error for error in errors))

    def test_extracts_documented_cli_paths_without_arguments(self):
        text = """
Run `stackcord projects --json`, then:
```sh
stackcord task apply --input request.json
stackcord harness release verify --root . --json
```
"""
        self.assertEqual(
            {("projects",), ("task", "apply"), ("harness", "release", "verify")},
            validate_docs.extract_stackcord_commands(text),
        )

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
