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
            self.assertIn(case["domain_command"], text)
            self.assertIn("context audit", text)
            self.assertEqual("context audit", case["first_command"])
            self.assertLessEqual(
                text.index(case["first_command"]),
                text.index(case["domain_command"]),
            )

    def test_behavior_surface_has_five_non_overlapping_skills(self):
        cases = json.loads((ROOT / "testdata" / "plugin" / "behavior.json").read_text())
        self.assertEqual(
            {
                "start-project",
                "continue-project",
                "plan-project-work",
                "coordinate-project-work",
                "recover-and-release-project",
            },
            {case["skill"] for case in cases},
        )


if __name__ == "__main__":
    unittest.main()
