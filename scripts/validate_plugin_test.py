import json
import pathlib
import re
import unittest

from validate_plugin import validate_hook_document


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

    def test_manifest_points_to_bundled_hooks(self):
        manifest = json.loads(
            (ROOT / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8")
        )
        self.assertEqual("./hooks/hooks.json", manifest.get("hooks"))

    def test_hooks_use_current_command_schema(self):
        hooks = json.loads(
            (ROOT / "hooks" / "hooks.json").read_text(encoding="utf-8")
        )
        self.assertEqual({"SessionStart", "PostCompact"}, set(hooks["hooks"]))
        for event in hooks["hooks"].values():
            command = event[0]["hooks"][0]
            self.assertEqual("command", command["type"])
            self.assertIn("orchestrator hook", command["command"])

    def test_agent_entry_links_exist(self):
        text = (ROOT / "AGENTS.md").read_text(encoding="utf-8")
        for path in re.findall(r"`([^`]+\.md)`", text):
            self.assertTrue((ROOT / path).is_file(), path)

    def test_hook_validator_accepts_command_hooks(self):
        value = json.loads(
            (ROOT / "testdata" / "plugin" / "hooks-valid.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertEqual([], validate_hook_document(value))

    def test_hook_validator_rejects_legacy_message_hooks(self):
        value = json.loads(
            (ROOT / "testdata" / "plugin" / "hooks-invalid.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertTrue(validate_hook_document(value))


if __name__ == "__main__":
    unittest.main()
