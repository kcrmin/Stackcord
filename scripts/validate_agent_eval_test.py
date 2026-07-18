import json
import pathlib
import tempfile
import unittest

from validate_agent_eval import load_document, validate
from run_agent_eval import build_codex_command, score_saved_transcript, score_transcript


ROOT = pathlib.Path(__file__).resolve().parents[1]


class AgentEvalContractTest(unittest.TestCase):
    def test_checked_in_evaluation_contract_is_valid(self):
        self.assertEqual([], validate(ROOT))

    def test_scenarios_cover_the_five_public_skills(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]
        self.assertEqual(
            {
                "start-project",
                "continue-project",
                "plan-project-work",
                "coordinate-project-work",
                "recover-and-release-project",
            },
            {scenario["expected_skill"] for scenario in scenarios},
        )

    def test_unknown_rubric_action_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / "evals/agent-behavior").mkdir(parents=True)
            (root / "skills/start-project").mkdir(parents=True)
            (root / "skills/start-project/SKILL.md").write_text(
                "---\nname: start-project\ndescription: Use when starting.\n---\n",
                encoding="utf-8",
            )
            (root / "evals/agent-behavior/scenarios.yaml").write_text(
                json.dumps({"version": 1, "scenarios": [{
                    "id": "bad", "prompt": "start", "expected_skill": "start-project",
                    "fixture": "new-project", "mode": "read-only",
                    "required_actions": ["missing"], "forbidden_actions": []
                }]}),
                encoding="utf-8",
            )
            (root / "evals/agent-behavior/rubric.yaml").write_text(
                json.dumps({"version": 1, "required_actions": {}, "forbidden_actions": {}}),
                encoding="utf-8",
            )
            self.assertTrue(any("unknown required action" in error for error in validate(root)))

    def test_runner_builds_ephemeral_least_privilege_codex_command(self):
        command = build_codex_command(
            executable="codex",
            fixture=pathlib.Path("/tmp/fixture"),
            mode="read-only",
            output=pathlib.Path("/tmp/final.txt"),
            prompt="continue",
            model="gpt-5.4",
        )
        self.assertEqual("codex", command[0])
        self.assertIn("--ephemeral", command)
        self.assertIn("read-only", command)
        self.assertIn("never", command)
        self.assertIn("gpt-5.4", command)
        self.assertLess(command.index("--model"), command.index("gpt-5.4"))
        self.assertNotIn("--dangerously-bypass-approvals-and-sandbox", command)

    def test_runner_scores_status_before_mutation_and_forbidden_content(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["combined_status_before_mutation", "one_safe_next_action"],
            "forbidden_actions": ["unsafe_mutation"],
        }
        passing = score_transcript(
            scenario,
            rubric,
            commands=["orchestrator status --json", "orchestrator work next --json"],
            response="확인된 상태를 기준으로 다음 안전한 작업을 제안합니다.",
        )
        self.assertTrue(passing["passed"])
        failing = score_transcript(
            scenario,
            rubric,
            commands=["git pull", "orchestrator status --json"],
            response="다음 작업입니다.",
        )
        self.assertFalse(failing["passed"])
        self.assertIn("combined_status_before_mutation", failing["missing_required"])
        self.assertIn("unsafe_mutation", failing["matched_forbidden"])

    def test_local_evaluation_transcripts_are_ignored(self):
        patterns = (ROOT / ".gitignore").read_text(encoding="utf-8").splitlines()
        self.assertIn(".harness/local/", patterns)

    def test_saved_transcript_can_be_rescored_without_model_execution(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "id": "saved",
            "expected_skill": "continue-project",
            "required_actions": ["combined_status_before_mutation", "one_safe_next_action"],
            "forbidden_actions": ["unsafe_mutation"],
        }
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            (output / "events.jsonl").write_text(
                json.dumps({"item": {"command": "orchestrator status --json"}}) + "\n",
                encoding="utf-8",
            )
            (output / "final.txt").write_text("이제 다음 작업을 진행합니다.\n", encoding="utf-8")
            result = score_saved_transcript(scenario, rubric, output)
            self.assertTrue(result["passed"])
            self.assertEqual("saved", result["id"])


if __name__ == "__main__":
    unittest.main()
