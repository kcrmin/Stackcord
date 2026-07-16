import json
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class PluginContractTest(unittest.TestCase):
    def test_manifest_and_marketplace_exist(self):
        self.assertTrue((ROOT / ".codex-plugin" / "plugin.json").is_file())
        self.assertTrue((ROOT / ".agents" / "plugins" / "marketplace.json").is_file())

    def test_every_behavior_has_a_focused_skill(self):
        cases = json.loads((ROOT / "testdata" / "plugin" / "behavior.json").read_text())
        for case in cases:
            skill = ROOT / "skills" / case["skill"] / "SKILL.md"
            self.assertTrue(skill.is_file(), case["skill"])
            text = skill.read_text()
            self.assertIn(f"name: {case['skill']}", text)
            self.assertIn(case["command"], text)
            self.assertIn("context audit", text)


if __name__ == "__main__":
    unittest.main()
