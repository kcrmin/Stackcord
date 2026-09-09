import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import unittest

import run_agent_eval
from validate_agent_eval import load_document, validate
from run_agent_eval import (
    build_codex_command,
    evaluation_environment,
    extract_web_searches,
    score_saved_transcript,
    score_transcript,
)


ROOT = pathlib.Path(__file__).resolve().parents[1]


class AgentEvalContractTest(unittest.TestCase):
    def test_agent_timeout_returns_a_scored_process_result(self):
        with tempfile.TemporaryDirectory() as directory:
            events = pathlib.Path(directory) / "events.jsonl"
            completed = run_agent_eval.execute_agent(
                [sys.executable, "-c", "import time; time.sleep(2)"],
                pathlib.Path(directory),
                dict(os.environ),
                events,
                timeout=0.01,
            )

            self.assertEqual(124, completed.returncode)
            self.assertIn("timed out", completed.stderr)

    def test_clean_clone_fixture_contains_a_readable_canonical_harness(self):
        go = shutil.which("go")
        if go is None:
            self.skipTest("Go is required for the real fixture integration test")
        scenario = {"id": "clean", "fixture": "clean-clone", "fixture_state": []}
        with tempfile.TemporaryDirectory() as directory:
            temporary = pathlib.Path(directory)
            cli = temporary / ("stackcord.exe" if os.name == "nt" else "stackcord")
            subprocess.run(
                [go, "build", "-trimpath", "-o", str(cli), "./cmd/stackcord"],
                cwd=ROOT / "cli",
                check=True,
            )
            fixture = temporary / "fixture"

            run_agent_eval._write_fixture(fixture, scenario, cli)

            self.assertTrue((fixture / ".harness" / "manifest.yaml").is_file())
            status = subprocess.run(
                [str(cli), "status", "--root", str(fixture), "--json"],
                text=True,
                stdout=subprocess.PIPE,
                check=False,
            )
            self.assertEqual(0, status.returncode, status.stdout)

    def test_fixture_uses_main_independent_of_the_machine_git_default(self):
        scenario = {"id": "clean", "fixture": "clean-clone", "fixture_state": []}
        with tempfile.TemporaryDirectory() as directory:
            fixture = pathlib.Path(directory) / "fixture"

            run_agent_eval._write_fixture(fixture, scenario)

            branch = subprocess.run(
                ["git", "-C", str(fixture), "branch", "--show-current"],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
            self.assertEqual("main", branch)

    def test_evaluation_workspace_is_created_below_the_ignored_output_root(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory) / ".harness" / "local" / "evals" / "drill"
            output.mkdir(parents=True)

            with run_agent_eval.evaluation_workspace(output) as workspace:
                self.assertTrue(workspace.is_relative_to(output))
                self.assertTrue(workspace.is_dir())

            self.assertFalse(workspace.exists())

    def test_fixture_stays_under_the_ignored_evaluation_workspace(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            with run_agent_eval.fixture_workspace(output, "clean-clone") as fixture:
                self.assertEqual(output / "clean-clone", fixture)

    def test_evaluation_cli_is_inside_fixture_without_dirtying_git_state(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            source = root / "stackcord.exe"
            source.write_bytes(b"fixture executable")
            fixture = root / "scenario"
            (fixture / ".git").mkdir(parents=True)
            cli = run_agent_eval.stage_fixture_cli(source, fixture)
            self.assertTrue(cli.is_relative_to(fixture))
            self.assertEqual(source.read_bytes(), cli.read_bytes())
            self.assertEqual(fixture / ".git" / "stackcord-eval" / source.name, cli)

    def test_checked_in_evaluation_contract_is_valid(self):
        self.assertEqual([], validate(ROOT))

    def test_scenarios_cover_the_six_public_skills(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]
        self.assertEqual(
            {
                "start-project",
                "continue-project",
                "plan-project-work",
                "coordinate-project-work",
                "recover-and-release-project",
                "use-git-conventions",
            },
            {scenario["expected_skill"] for scenario in scenarios},
        )

    def test_tool_selection_scenario_requires_current_official_comparison(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]
        scenario = next(
            item for item in scenarios if item["id"] == "current-tool-selection"
        )

        self.assertIn("inspect_available_tools", scenario["required_actions"])
        self.assertIn("verify_current_official_evidence", scenario["required_actions"])
        self.assertIn("compare_realistic_candidates", scenario["required_actions"])
        self.assertIn("ask_one_material_question", scenario["required_actions"])
        self.assertIn("install_without_selection", scenario["forbidden_actions"])

    def test_runner_requires_an_explicit_scenario_or_all_opt_in(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]

        with self.assertRaisesRegex(ValueError, "select one to three scenarios or pass --all"):
            run_agent_eval.select_scenarios(
                scenarios,
                requested=[],
                run_all=False,
                allow_external_research=False,
            )

        selected = run_agent_eval.select_scenarios(
            scenarios,
            requested=["continue-after-clean-clone"],
            run_all=False,
            allow_external_research=False,
        )
        self.assertEqual(["continue-after-clean-clone"], [item["id"] for item in selected])

    def test_full_suite_and_external_research_need_separate_opt_ins(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]

        with self.assertRaisesRegex(ValueError, "external tool research"):
            run_agent_eval.select_scenarios(
                scenarios,
                requested=[],
                run_all=True,
                allow_external_research=False,
            )
        selected = run_agent_eval.select_scenarios(
            scenarios,
            requested=[],
            run_all=True,
            allow_external_research=True,
        )
        self.assertEqual(10, len(selected))

    def test_normal_selection_is_limited_to_three_and_research_is_not_implicit(self):
        scenarios = load_document(ROOT / "evals/agent-behavior/scenarios.yaml")["scenarios"]

        with self.assertRaisesRegex(ValueError, "at most three"):
            run_agent_eval.select_scenarios(
                scenarios,
                requested=[item["id"] for item in scenarios[:4]],
                run_all=False,
                allow_external_research=False,
            )
        with self.assertRaisesRegex(ValueError, "external tool research"):
            run_agent_eval.select_scenarios(
                scenarios,
                requested=["current-tool-selection"],
                run_all=False,
                allow_external_research=False,
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
        self.assertNotIn('sandbox_permissions=["disk-full-read-access"]', command)
        self.assertIn("never", command)
        self.assertIn("gpt-5.4", command)
        self.assertNotIn("--add-dir", command)
        self.assertLess(command.index("--model"), command.index("gpt-5.4"))
        self.assertNotIn("--dangerously-bypass-approvals-and-sandbox", command)

    def test_runner_exposes_the_built_cli_without_changing_user_path(self):
        base = {"PATH": "/usr/bin", "HOME": "/tmp/home"}
        cli = pathlib.Path("/tmp/eval-bin/stackcord")

        environment = evaluation_environment(base, cli)

        self.assertEqual(str(cli), environment["STACKCORD_CLI"])
        self.assertEqual(f"{cli.parent}{os.pathsep}/usr/bin", environment["PATH"])
        self.assertEqual("0", environment["GIT_TERMINAL_PROMPT"])
        self.assertEqual({"PATH": "/usr/bin", "HOME": "/tmp/home"}, base)

    def test_runner_scores_status_before_mutation_and_forbidden_content(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["combined_status_before_mutation", "one_safe_next_action"],
            "forbidden_actions": ["unsafe_mutation"],
        }
        passing = score_transcript(
            scenario,
            rubric,
            commands=["stackcord status --json", "stackcord work next --json"],
            response="확인된 상태를 기준으로 다음 안전한 작업을 제안합니다.",
        )
        self.assertTrue(passing["passed"])
        failing = score_transcript(
            scenario,
            rubric,
            commands=["git pull", "stackcord status --json"],
            response="다음 작업입니다.",
        )
        self.assertFalse(failing["passed"])
        self.assertIn("combined_status_before_mutation", failing["missing_required"])
        self.assertIn("unsafe_mutation", failing["matched_forbidden"])

    def test_configured_cli_environment_invocation_scores_as_stackcord(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["combined_status_before_mutation"],
            "forbidden_actions": [],
        }

        result = score_transcript(
            scenario,
            rubric,
            commands=["& $env:STACKCORD_CLI status --json"],
            response="상태를 확인했습니다.",
        )

        self.assertTrue(result["passed"])

    def test_help_before_status_is_not_scored_as_a_mutation(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["combined_status_before_mutation"],
            "forbidden_actions": [],
        }

        result = score_transcript(
            scenario,
            rubric,
            commands=[
                "stackcord project checkpoint --help",
                "stackcord status --json",
            ],
            response="상태를 확인했습니다.",
        )

        self.assertTrue(result["passed"])

    def test_current_official_evidence_requires_an_observed_web_search(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["verify_current_official_evidence"],
            "forbidden_actions": [],
        }
        response = "현재 공식 문서와 최신 릴리스 상태를 확인했습니다."

        claimed = score_transcript(
            scenario,
            rubric,
            commands=[],
            response=response,
            web_searches=[],
        )
        observed = score_transcript(
            scenario,
            rubric,
            commands=[],
            response=response,
            web_searches=["https://example.invalid/official"],
        )

        self.assertFalse(claimed["passed"])
        self.assertTrue(observed["passed"])

    def test_web_search_events_are_extracted_from_codex_jsonl(self):
        with tempfile.TemporaryDirectory() as directory:
            events = pathlib.Path(directory) / "events.jsonl"
            events.write_text(
                json.dumps({
                    "type": "item.completed",
                    "item": {
                        "type": "web_search",
                        "query": "official current release status",
                        "action": {"url": "https://example.invalid/releases"},
                    },
                }) + "\n",
                encoding="utf-8",
            )

            searches = extract_web_searches(events)

            self.assertEqual(1, len(searches))
            self.assertIn("official current release status", searches[0])

    def test_proportional_coordination_accepts_natural_korean_wording(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["proportional_coordination"],
            "forbidden_actions": ["unnecessary_task_gate"],
        }

        result = score_transcript(
            scenario,
            rubric,
            commands=[],
            response="이 작은 문서 수정에는 티켓, 예약, TDD 같은 조율 절차가 필요 없습니다.",
        )

        self.assertTrue(result["passed"])

    def test_provider_boundary_accepts_natural_korean_live_state_wording(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["provider_truth_boundary"],
            "forbidden_actions": [],
        }

        result = score_transcript(
            scenario,
            rubric,
            commands=[],
            response="하네스가 없어 라이브 작업 상태는 이 클론에서 복구되지 않습니다.",
        )

        self.assertTrue(result["passed"])

    def test_contract_first_accepts_approval_before_workspace_slices(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["contract_first_resolution"],
            "forbidden_actions": [],
        }

        result = score_transcript(
            scenario,
            rubric,
            commands=[],
            response="공통 규칙을 정하고 API 계약 승인 후 UI와 API 작업을 나눕니다.",
        )

        self.assertTrue(result["passed"])

    def test_failed_checkpoint_command_does_not_prove_persisted_product_meaning(self):
        rubric = load_document(ROOT / "evals/agent-behavior/rubric.yaml")
        scenario = {
            "required_actions": ["checkpoint_normalized_product_meaning"],
            "forbidden_actions": [],
        }
        command = "stackcord project checkpoint --input checkpoint.yaml --apply --json"

        failed = score_transcript(
            scenario,
            rubric,
            commands=[command],
            successful_commands=[],
            response="정규화한 제품 의도를 정리해 저장했습니다.",
        )
        passed = score_transcript(
            scenario,
            rubric,
            commands=[command],
            successful_commands=[command],
            response="정규화한 제품 의도를 저장했습니다.",
        )

        self.assertFalse(failed["passed"])
        self.assertTrue(passed["passed"])

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
                json.dumps({"item": {"command": "stackcord status --json"}}) + "\n",
                encoding="utf-8",
            )
            (output / "final.txt").write_text("이제 다음 작업을 진행합니다.\n", encoding="utf-8")
            result = score_saved_transcript(scenario, rubric, output)
            self.assertTrue(result["passed"])
            self.assertEqual("saved", result["id"])


if __name__ == "__main__":
    unittest.main()
