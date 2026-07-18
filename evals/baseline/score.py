#!/usr/bin/env python3
"""Score deterministic dogfood observations against a declared manual baseline."""

from __future__ import annotations

import argparse
import json
import pathlib
from typing import Any


def load_document(path: pathlib.Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as error:
        raise ValueError(f"missing baseline input: {path}") from error
    except json.JSONDecodeError as error:
        raise ValueError(f"invalid JSON-compatible YAML {path}: {error}") from error
    if not isinstance(value, dict):
        raise ValueError(f"baseline input must be an object: {path}")
    return value


def score_result(scenarios_document: dict[str, Any], dogfood: dict[str, Any]) -> dict[str, Any]:
    if scenarios_document.get("schema_version") != 1:
        raise ValueError("baseline scenario schema_version must be 1")
    if scenarios_document.get("baseline") != "manual-git-and-static-docs":
        raise ValueError("baseline must be manual-git-and-static-docs")
    scenarios = scenarios_document.get("scenarios")
    if not isinstance(scenarios, list) or not scenarios:
        raise ValueError("baseline scenarios must be a non-empty list")
    if dogfood.get("schema_version") != 1:
        raise ValueError("dogfood schema_version must be 1")
    assertion_records = dogfood.get("assertions")
    if not isinstance(assertion_records, list):
        raise ValueError("dogfood assertions must be a list")

    assertions: dict[str, str] = {}
    for record in assertion_records:
        if not isinstance(record, dict):
            raise ValueError("dogfood assertion must be an object")
        code, status = record.get("code"), record.get("status")
        if not isinstance(code, str) or status not in {"passed", "failed"}:
            raise ValueError("dogfood assertion code or status is invalid")
        if code in assertions:
            raise ValueError(f"duplicate dogfood assertion: {code}")
        assertions[code] = status
    measurements = dogfood.get("measurements", {})
    observed_failing_tests = measurements.get("observed_failing_tests", 0) if isinstance(measurements, dict) else 0
    if not isinstance(observed_failing_tests, int) or observed_failing_tests < 0:
        raise ValueError("dogfood observed_failing_tests must be a non-negative integer")

    seen: set[str] = set()
    rows: list[dict[str, str]] = []
    harness_detected = 0
    manual_deterministic = 0
    for scenario in scenarios:
        if not isinstance(scenario, dict):
            raise ValueError("baseline scenario must be an object")
        scenario_id = scenario.get("id")
        required = scenario.get("harness_assertions")
        note = scenario.get("manual_note")
        manual = scenario.get("manual_deterministic")
        if not isinstance(scenario_id, str) or not scenario_id or scenario_id in seen:
            raise ValueError(f"baseline scenario id is invalid or duplicated: {scenario_id!r}")
        seen.add(scenario_id)
        if not isinstance(required, list) or not required or not all(isinstance(code, str) and code for code in required):
            raise ValueError(f"baseline scenario {scenario_id} needs harness assertions")
        if not isinstance(manual, bool) or not isinstance(note, str) or not note.strip():
            raise ValueError(f"baseline scenario {scenario_id} needs a bounded manual comparison")
        detected = all(assertions.get(code) == "passed" for code in required)
        if detected:
            harness_detected += 1
        if manual:
            manual_deterministic += 1
        rows.append({
            "id": scenario_id,
            "harness_result": "detected" if detected else "missed",
            "manual_result": "deterministic-check" if manual else "not-deterministic",
            "manual_note": note.strip(),
        })

    count = len(rows)
    return {
        "schema_version": 1,
        "passed": dogfood.get("status") == "passed" and harness_detected == count,
        "scenario_count": count,
        "harness": {"detected": harness_detected, "missed": count - harness_detected},
        "manual_baseline": {
            "deterministic_checks": manual_deterministic,
            "not_deterministically_covered": count - manual_deterministic,
        },
        "dogfood": {
            "passed": sum(status == "passed" for status in assertions.values()),
            "failed": sum(status == "failed" for status in assertions.values()),
            "observed_failing_tests": observed_failing_tests,
        },
        "scenarios": rows,
    }


def _cell(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ")


def write_markdown(report: dict[str, Any], target: pathlib.Path) -> None:
    count = int(report["scenario_count"])
    detected = int(report["harness"]["detected"])
    manual = int(report["manual_baseline"]["deterministic_checks"])
    dogfood_passed = int(report["dogfood"]["passed"])
    dogfood_failed = int(report["dogfood"]["failed"])
    red_tests = int(report["dogfood"]["observed_failing_tests"])
    lines = [
        "# Multi-repository dogfood report",
        "",
        "This report compares deterministic checks, not team productivity. "
        "No elapsed-time or human-performance claim is made from this local fixture.",
        "",
        f"- Harness observations: **{detected}/{count}** scenarios detected",
        f"- Manual Git + static docs baseline: **{manual}/{count}** scenarios have a deterministic native check",
        f"- Dogfood assertions: **{dogfood_passed}/{dogfood_passed + dogfood_failed}** passed",
        f"- TDD proof: **{red_tests} expected failing test runs** were observed before their implementations passed",
        f"- Result: **{'PASS' if report.get('passed') else 'FAIL'}**",
        "",
        "| Scenario | Harness | Manual Git + static docs | Boundary |",
        "|---|---:|---:|---|",
    ]
    for row in report["scenarios"]:
        lines.append(
            f"| `{_cell(row['id'])}` | {row['harness_result']} | {row['manual_result']} | {_cell(row['manual_note'])} |"
        )
    lines.extend([
        "",
        "The manual column does not mean a careful engineer cannot discover the problem. "
        "It records whether ordinary Git plus static documentation provides a deterministic, service-aware check without this harness.",
        "",
        "The dogfood fixture uses local bare remotes and public-looking placeholder URLs. "
        "It proves repository behavior without claiming hosted GitHub or Jira writes, network performance, or production load capacity.",
        "",
    ])
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("\n".join(lines), encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scenarios", required=True, type=pathlib.Path)
    parser.add_argument("--dogfood", required=True, type=pathlib.Path)
    parser.add_argument("--output-json", type=pathlib.Path)
    parser.add_argument("--output-markdown", type=pathlib.Path)
    args = parser.parse_args()
    try:
        report = score_result(load_document(args.scenarios), load_document(args.dogfood))
    except ValueError as error:
        parser.error(str(error))
    if args.output_json:
        args.output_json.parent.mkdir(parents=True, exist_ok=True)
        args.output_json.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    if args.output_markdown:
        write_markdown(report, args.output_markdown)
    print(json.dumps(report, sort_keys=True))
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
