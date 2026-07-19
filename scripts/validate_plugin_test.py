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
            self.assertIn("orchestrator status --json", text)
            self.assertEqual("orchestrator status --json", case["first_command"])
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

    def test_work_planning_is_proportional_not_a_universal_gate(self):
        text = (ROOT / "skills" / "plan-project-work" / "SKILL.md").read_text(
            encoding="utf-8"
        )
        self.assertIn("small private", text)
        self.assertNotIn("then reserve it before creating implementation state", text)

    def test_material_questions_use_the_user_facing_choice_contract(self):
        for name in ("start-project", "plan-project-work"):
            text = (ROOT / "skills" / name / "SKILL.md").read_text(encoding="utf-8")
            self.assertIn("A/B/C", text)
            self.assertIn("free-form", text)

    def test_discovery_persists_initial_intent_before_the_next_question(self):
        text = (ROOT / "skills" / "start-project" / "SKILL.md").read_text(
            encoding="utf-8"
        )
        self.assertIn("initial product request", text)
        self.assertIn("successful apply", text)

    def test_discovery_defers_project_setup_until_the_boundary_is_known(self):
        text = (ROOT / "skills" / "start-project" / "SKILL.md").read_text(
            encoding="utf-8"
        )
        self.assertIn("existing product files already establish", text)
        self.assertIn("first high-impact scope answer", text)

    def test_ui_tools_are_optional_inputs_to_an_editable_baseline(self):
        start = (ROOT / "skills" / "start-project" / "SKILL.md").read_text(
            encoding="utf-8"
        )
        coordinate = (
            ROOT / "skills" / "coordinate-project-work" / "SKILL.md"
        ).read_text(encoding="utf-8")
        continue_text = (
            ROOT / "skills" / "continue-project" / "SKILL.md"
        ).read_text(encoding="utf-8")
        self.assertIn("MengTo/Skills", start)
        self.assertIn("optional UI creation", start)
        self.assertIn("ordinary editable files", coordinate)
        self.assertIn("orchestrator ui promote", coordinate)
        self.assertIn("orchestrator ui baseline bind", coordinate)
        self.assertIn("exact UI baseline", continue_text)
        self.assertNotIn("MengTo/Skills is the source of truth", start)

    def test_protected_product_meaning_requires_real_product_authority(self):
        coordinate = (
            ROOT / "skills" / "coordinate-project-work" / "SKILL.md"
        ).read_text(encoding="utf-8")
        release = (
            ROOT / "skills" / "recover-and-release-project" / "SKILL.md"
        ).read_text(encoding="utf-8")
        self.assertIn("product authority", coordinate.lower())
        self.assertIn("orchestrator governance check --json", coordinate)
        self.assertIn("Git display name and email", coordinate)
        self.assertIn("may not present it as approved", coordinate)
        self.assertIn("fresh product-authority approval", release)

    def test_manifest_points_to_bundled_hooks(self):
        manifest = json.loads(
            (ROOT / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8")
        )
        self.assertEqual("./hooks/hooks.json", manifest.get("hooks"))

    def test_manifest_names_the_service_continuity_differentiator(self):
        manifest = json.loads(
            (ROOT / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8")
        )
        interface = manifest["interface"]
        combined = " ".join(
            [manifest["description"], interface["longDescription"], *interface["capabilities"]]
        ).lower()
        for token in (
            "durable context",
            "multi-repository",
            "semantic work reservation",
            "external task",
            "exact release",
        ):
            self.assertIn(token, combined)

    def test_hooks_use_current_command_schema(self):
        hooks = json.loads(
            (ROOT / "hooks" / "hooks.json").read_text(encoding="utf-8")
        )
        self.assertEqual({"SessionStart", "PostCompact"}, set(hooks["hooks"]))
        for event in hooks["hooks"].values():
            command = event[0]["hooks"][0]
            self.assertEqual("command", command["type"])
            self.assertIn("$PLUGIN_ROOT/hooks/run-orchestrator-hook.sh", command["command"])
            self.assertIn("$env:PLUGIN_ROOT", command["commandWindows"])
            self.assertIn("run-orchestrator-hook.ps1", command["commandWindows"])
        shell_resolver = (ROOT / "hooks" / "run-orchestrator-hook.sh").read_text(
            encoding="utf-8"
        )
        powershell_resolver = (
            ROOT / "hooks" / "run-orchestrator-hook.ps1"
        ).read_text(encoding="utf-8")
        self.assertIn('exec "$CLI" hook "$EVENT"', shell_resolver)
        self.assertIn("& $Cli hook $Event", powershell_resolver)

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
