#!/usr/bin/env python3
"""Tests for the raw scenario baseline scorer."""

from __future__ import annotations

import importlib.util
import pathlib
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("score.py")


def load_score_module():
    spec = importlib.util.spec_from_file_location("baseline_score", MODULE_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError("baseline scorer cannot be loaded")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class BaselineScoreTest(unittest.TestCase):
    def test_scores_only_observed_harness_assertions_and_declared_manual_checks(self) -> None:
        score = load_score_module()
        scenarios = {
            "schema_version": 1,
            "baseline": "manual-git-and-static-docs",
            "scenarios": [
                {
                    "id": "semantic-conflict",
                    "harness_assertions": ["conflict.detected"],
                    "manual_deterministic": False,
                    "manual_note": "Git has no service semantic model.",
                },
                {
                    "id": "pointer-drift",
                    "harness_assertions": ["pointer.detected", "pointer.safe"],
                    "manual_deterministic": True,
                    "manual_note": "Native Git can show a changed gitlink.",
                },
            ],
        }
        dogfood = {
            "schema_version": 1,
            "status": "failed",
            "assertions": [
                {"code": "conflict.detected", "status": "passed"},
                {"code": "pointer.detected", "status": "passed"},
                {"code": "pointer.safe", "status": "failed"},
            ],
        }

        report = score.score_result(scenarios, dogfood)

        self.assertEqual(2, report["scenario_count"])
        self.assertEqual({"detected": 1, "missed": 1}, report["harness"])
        self.assertEqual(
            {"deterministic_checks": 1, "not_deterministically_covered": 1},
            report["manual_baseline"],
        )
        self.assertEqual({"passed": 2, "failed": 1, "observed_failing_tests": 0}, report["dogfood"])
        self.assertFalse(report["passed"])
        self.assertEqual("missed", report["scenarios"][1]["harness_result"])

    def test_rejects_duplicate_or_unknown_assertion_records(self) -> None:
        score = load_score_module()
        scenarios = {
            "schema_version": 1,
            "baseline": "manual-git-and-static-docs",
            "scenarios": [{
                "id": "clone",
                "harness_assertions": ["clone.recovered"],
                "manual_deterministic": False,
                "manual_note": "No combined state model.",
            }],
        }
        dogfood = {
            "schema_version": 1,
            "status": "passed",
            "assertions": [
                {"code": "clone.recovered", "status": "passed"},
                {"code": "clone.recovered", "status": "passed"},
            ],
        }

        with self.assertRaisesRegex(ValueError, "duplicate dogfood assertion"):
            score.score_result(scenarios, dogfood)

    def test_markdown_states_that_timing_and_human_performance_are_not_measured(self) -> None:
        score = load_score_module()
        report = {
            "schema_version": 1,
            "passed": True,
            "scenario_count": 1,
            "harness": {"detected": 1, "missed": 0},
            "manual_baseline": {"deterministic_checks": 0, "not_deterministically_covered": 1},
            "dogfood": {"passed": 3, "failed": 0, "observed_failing_tests": 2},
            "scenarios": [{
                "id": "clone",
                "harness_result": "detected",
                "manual_result": "not-deterministic",
                "manual_note": "No combined state model.",
            }],
        }

        with tempfile.TemporaryDirectory() as directory:
            target = pathlib.Path(directory) / "report.md"
            score.write_markdown(report, target)
            content = target.read_text(encoding="utf-8")

        self.assertIn("1/1", content)
        self.assertIn("3/3", content)
        self.assertIn("2 expected failing test runs", content)
        self.assertIn("No elapsed-time or human-performance claim", content)


if __name__ == "__main__":
    unittest.main()
