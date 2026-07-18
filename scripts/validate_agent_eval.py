#!/usr/bin/env python3
"""Validate checked-in agent behavior scenarios without third-party packages."""

from __future__ import annotations

import json
import pathlib
import re
import sys
from typing import Any


IDENTIFIER = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
ACTION_IDENTIFIER = re.compile(r"^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$")
ALLOWED_MODES = {"read-only", "workspace-write"}
REQUIRED_SCENARIOS = {
    "new-project-discovery",
    "continue-after-clean-clone",
    "recover-after-context-loss",
    "cross-repo-semantic-conflict",
    "selected-provider-unavailable",
    "local-only-work",
    "release-candidate-mismatch",
}
REQUIRED_SKILLS = {
    "start-project",
    "continue-project",
    "plan-project-work",
    "coordinate-project-work",
    "recover-and-release-project",
}


def load_document(path: pathlib.Path) -> Any:
    """Load JSON-compatible YAML used by the dependency-free release validator."""

    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as error:
        raise ValueError(f"missing evaluation document: {path}") from error
    except json.JSONDecodeError as error:
        raise ValueError(
            f"{path} must be JSON-compatible YAML: line {error.lineno}, column {error.colno}"
        ) from error


def _validate_action_map(errors: list[str], label: str, value: object) -> set[str]:
    if not isinstance(value, dict):
        errors.append(f"rubric {label} must be an object")
        return set()
    names: set[str] = set()
    for name, rule in value.items():
        if not isinstance(name, str) or not ACTION_IDENTIFIER.fullmatch(name):
            errors.append(f"rubric {label} has invalid action id {name!r}")
            continue
        names.add(name)
        if not isinstance(rule, dict):
            errors.append(f"rubric action {name} must be an object")
            continue
        patterns = rule.get("patterns")
        if not isinstance(patterns, list) or not patterns or not all(
            isinstance(pattern, str) and pattern.strip() for pattern in patterns
        ):
            errors.append(f"rubric action {name} needs non-empty string patterns")
        if label == "required_actions" and rule.get("kind") not in {
            "command_before_mutation",
            "command_or_response",
            "response",
        }:
            errors.append(f"rubric action {name} has unsupported kind")
    return names


def validate(root: pathlib.Path) -> list[str]:
    errors: list[str] = []
    eval_dir = root / "evals" / "agent-behavior"
    try:
        scenarios_doc = load_document(eval_dir / "scenarios.yaml")
        rubric_doc = load_document(eval_dir / "rubric.yaml")
    except ValueError as error:
        return [str(error)]

    if not isinstance(scenarios_doc, dict) or scenarios_doc.get("version") != 1:
        errors.append("scenario document version must be 1")
        scenarios: object = None
    else:
        scenarios = scenarios_doc.get("scenarios")
    if not isinstance(rubric_doc, dict) or rubric_doc.get("version") != 1:
        errors.append("rubric document version must be 1")

    required = _validate_action_map(
        errors, "required_actions", rubric_doc.get("required_actions") if isinstance(rubric_doc, dict) else None
    )
    forbidden = _validate_action_map(
        errors, "forbidden_actions", rubric_doc.get("forbidden_actions") if isinstance(rubric_doc, dict) else None
    )

    if not isinstance(scenarios, list) or not scenarios:
        errors.append("scenarios must be a non-empty list")
        return errors

    seen_ids: set[str] = set()
    seen_skills: set[str] = set()
    for index, scenario in enumerate(scenarios):
        label = f"scenario[{index}]"
        if not isinstance(scenario, dict):
            errors.append(f"{label} must be an object")
            continue
        scenario_id = scenario.get("id")
        if not isinstance(scenario_id, str) or not IDENTIFIER.fullmatch(scenario_id):
            errors.append(f"{label} has invalid id")
        elif scenario_id in seen_ids:
            errors.append(f"duplicate scenario id {scenario_id}")
        else:
            seen_ids.add(scenario_id)
        prompt = scenario.get("prompt")
        if not isinstance(prompt, str) or not prompt.strip():
            errors.append(f"{label} needs a prompt")
        skill = scenario.get("expected_skill")
        if not isinstance(skill, str) or not (root / "skills" / skill / "SKILL.md").is_file():
            errors.append(f"{label} references missing skill {skill!r}")
        else:
            seen_skills.add(skill)
        if scenario.get("mode") not in ALLOWED_MODES:
            errors.append(f"{label} has unsupported mode")
        if not isinstance(scenario.get("fixture"), str) or not scenario["fixture"].strip():
            errors.append(f"{label} needs a fixture")
        for field, known in (("required_actions", required), ("forbidden_actions", forbidden)):
            actions = scenario.get(field)
            if not isinstance(actions, list) or not actions or not all(isinstance(item, str) for item in actions):
                errors.append(f"{label} {field} must be a non-empty string list")
                continue
            for action in actions:
                if action not in known:
                    action_type = "required action" if field == "required_actions" else "forbidden action"
                    errors.append(f"{label} has unknown {action_type} {action}")

    if root.resolve() == pathlib.Path(__file__).resolve().parents[1]:
        missing = REQUIRED_SCENARIOS - seen_ids
        if missing:
            errors.append(f"missing required scenarios: {sorted(missing)}")
        if seen_skills != REQUIRED_SKILLS:
            errors.append(
                f"evaluation skill coverage differs: expected {sorted(REQUIRED_SKILLS)}, got {sorted(seen_skills)}"
            )
    return errors


def main() -> int:
    root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = validate(root)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Agent evaluation validation passed: {root}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
