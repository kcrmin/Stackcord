import unittest
from ci_scope import needs_full_tests, gate_passes


class ScopeTest(unittest.TestCase):
    def test_only_documentation_can_skip_platform_jobs(self):
        self.assertFalse(needs_full_tests(["README.md", "docs/concepts/ko.md"]))
        for path in ("cli/a.go", "skills/start-project/SKILL.md", "schemas/a.json",
                     "scripts/x.py", ".github/workflows/ci.yml", "new-area/readme.md"):
            self.assertTrue(needs_full_tests(["README.md", path]), path)
        self.assertTrue(needs_full_tests([]))

    def test_gate_requires_success_and_only_expected_skips(self):
        states = {"changes": "success", "repository-contracts": "success",
                  "native": "success", "product-dogfood": "success", "cross-build": "success"}
        self.assertTrue(gate_passes(True, states))
        for result in ("failure", "cancelled", "skipped", "unknown", ""):
            self.assertFalse(gate_passes(True, dict(states, native=result)))
        light = dict(states, native="skipped", **{"product-dogfood": "skipped", "cross-build": "skipped"})
        self.assertTrue(gate_passes(False, light))
        self.assertFalse(gate_passes(False, dict(light, **{"repository-contracts": "skipped"})))
