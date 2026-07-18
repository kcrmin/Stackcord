#!/usr/bin/env python3
"""Validate the distributable plugin without third-party dependencies."""

import json
import pathlib
import re
import sys


NAME = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
SEMVER = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?$")
LINK = re.compile(r"\[[^]]+\]\(([^)]+)\)")
HOOK_EVENTS = ("SessionStart", "PostCompact")
EXPECTED_SKILLS = {
    "start-project",
    "continue-project",
    "plan-project-work",
    "coordinate-project-work",
    "recover-and-release-project",
}


def fail(errors, message):
    errors.append(message)


def validate_hook_document(value: object) -> list[str]:
    errors: list[str] = []
    if not isinstance(value, dict) or not isinstance(value.get("hooks"), dict):
        return ["hooks must be an event-keyed object"]
    hooks = value["hooks"]
    for name in HOOK_EVENTS:
        groups = hooks.get(name)
        if not isinstance(groups, list) or not groups:
            errors.append(f"missing hook event {name}")
            continue
        for group in groups:
            if not isinstance(group, dict):
                errors.append(f"{name} matcher group must be an object")
                continue
            commands = group.get("hooks")
            if not isinstance(commands, list) or not commands:
                errors.append(f"{name} must contain command hooks")
                continue
            for command in commands:
                if (
                    not isinstance(command, dict)
                    or command.get("type") != "command"
                    or not isinstance(command.get("command"), str)
                    or not command["command"].strip()
                ):
                    errors.append(f"{name} must contain valid command hooks")
    return errors


def validate(root: pathlib.Path) -> list[str]:
    errors: list[str] = []
    manifest_dir = root / ".codex-plugin"
    manifest_path = manifest_dir / "plugin.json"
    files = sorted(path for path in manifest_dir.glob("*") if path.is_file()) if manifest_dir.is_dir() else []
    if files != [manifest_path]:
        fail(errors, ".codex-plugin must contain only plugin.json")
        return errors
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except Exception as error:
        fail(errors, f"invalid plugin manifest: {error}")
        return errors
    for field in ("name", "version", "description", "author", "skills", "interface"):
        if field not in manifest:
            fail(errors, f"manifest missing {field}")
    if not NAME.fullmatch(str(manifest.get("name", ""))):
        fail(errors, "manifest name is not kebab-case")
    if not SEMVER.fullmatch(str(manifest.get("version", ""))):
        fail(errors, "manifest version is not strict semver")
    if manifest.get("hooks") != "./hooks/hooks.json":
        fail(errors, "manifest must point to ./hooks/hooks.json")
    if "[TODO:" in manifest_path.read_text(encoding="utf-8"):
        fail(errors, "manifest contains TODO placeholders")
    interface = manifest.get("interface", {})
    for field in ("displayName", "shortDescription", "longDescription", "developerName", "category", "capabilities", "defaultPrompt"):
        if not interface.get(field):
            fail(errors, f"manifest interface missing {field}")

    expected_names = sorted(EXPECTED_SKILLS)
    skill_dirs = sorted(path for path in (root / "skills").iterdir() if path.is_dir())
    actual_names = sorted(path.name for path in skill_dirs)
    if actual_names != expected_names:
        fail(errors, f"skill set differs: expected {expected_names}, got {actual_names}")
    for directory in skill_dirs:
        skill_file = directory / "SKILL.md"
        if not skill_file.is_file():
            fail(errors, f"{directory.name} has no SKILL.md")
            continue
        text = skill_file.read_text(encoding="utf-8")
        frontmatter = re.match(r"^---\nname: ([^\n]+)\ndescription: ([^\n]+)\n---\n", text)
        if not frontmatter:
            fail(errors, f"{directory.name} has invalid frontmatter")
            continue
        if frontmatter.group(1) != directory.name:
            fail(errors, f"{directory.name} frontmatter name differs")
        if not frontmatter.group(2).startswith("Use when"):
            fail(errors, f"{directory.name} description must begin with Use when")
        if "orchestrator status --json" not in text:
            fail(errors, f"{directory.name} does not recover combined status")
        if "[TODO:" in text:
            fail(errors, f"{directory.name} contains TODO placeholders")
        if "classes:\n" in text or "main_protected:" in text:
            fail(errors, f"{directory.name} duplicates canonical policy")
        for link in LINK.findall(text):
            if "://" in link or link.startswith("#"):
                continue
            target = (directory / link).resolve()
            try:
                target.relative_to(root.resolve())
            except ValueError:
                fail(errors, f"{directory.name} reference escapes plugin: {link}")
                continue
            if not target.exists():
                fail(errors, f"{directory.name} reference missing: {link}")

    marketplace = json.loads((root / ".agents" / "plugins" / "marketplace.json").read_text(encoding="utf-8"))
    entry = marketplace.get("plugins", [{}])[0]
    if entry.get("name") != manifest.get("name"):
        fail(errors, "marketplace and manifest names differ")
    if entry.get("policy", {}).get("installation") not in {"NOT_AVAILABLE", "AVAILABLE", "INSTALLED_BY_DEFAULT"}:
        fail(errors, "marketplace installation policy is invalid")
    if entry.get("policy", {}).get("authentication") not in {"ON_INSTALL", "ON_USE"}:
        fail(errors, "marketplace authentication policy is invalid")
    if not entry.get("category"):
        fail(errors, "marketplace category is required")

    hook_path = root / "hooks" / "hooks.json"
    try:
        hooks = json.loads(hook_path.read_text(encoding="utf-8"))
    except Exception as error:
        fail(errors, f"invalid hook document: {error}")
    else:
        errors.extend(validate_hook_document(hooks))
    return errors


def main() -> int:
    root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = validate(root)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Plugin validation passed: {root}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
