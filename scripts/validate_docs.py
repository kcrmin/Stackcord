#!/usr/bin/env python3
"""Check English/Korean document pairing and catalog parity."""

import json
import pathlib
import re
import sys


ROOT = pathlib.Path(__file__).resolve().parents[1]
PAIRS = [
    ("docs/getting-started/en.md", "docs/getting-started/ko.md"),
    ("docs/concepts/en.md", "docs/concepts/ko.md"),
    *[(f"docs/guides/{name}-en.md", f"docs/guides/{name}-ko.md") for name in ("new-project", "existing-project", "submodules", "dbdiagram", "release", "troubleshooting")],
    *[(f"docs/security/{name}-en.md", f"docs/security/{name}-ko.md") for name in ("threat-model", "privacy")],
]


def headings(text: str) -> list[int]:
    return [len(match.group(1)) for match in re.finditer(r"^(#+) ", text, re.MULTILINE)]


def validate() -> list[str]:
    errors: list[str] = []
    for english_path, korean_path in PAIRS:
        english_file, korean_file = ROOT / english_path, ROOT / korean_path
        if not english_file.is_file() or not korean_file.is_file():
            errors.append(f"missing document pair: {english_path} / {korean_path}")
            continue
        english, korean = english_file.read_text(encoding="utf-8"), korean_file.read_text(encoding="utf-8")
        if headings(english) != headings(korean):
            errors.append(f"heading structure differs: {english_path} / {korean_path}")
    for locale in ("en", "ko"):
        public = json.loads((ROOT / "locales" / locale / "messages.json").read_text(encoding="utf-8"))
        embedded = json.loads((ROOT / "cli" / "internal" / "output" / "catalogs" / f"{locale}.json").read_text(encoding="utf-8"))
        if public != embedded:
            errors.append(f"public and embedded {locale} catalogs differ")
    obsolete = ["product itself is not implemented", "제품 자체는 아직 구현"]
    readmes = (ROOT / "README.md").read_text(encoding="utf-8") + (ROOT / "README.ko.md").read_text(encoding="utf-8")
    for phrase in obsolete:
        if phrase in readmes:
            errors.append(f"obsolete status remains: {phrase}")

    public_files = [ROOT / "README.md", ROOT / "README.ko.md"]
    for directory in ("docs/getting-started", "docs/concepts", "docs/guides", "docs/security", "skills", "references"):
        public_files.extend((ROOT / directory).rglob("*.md"))
    public_text = "\n".join(path.read_text(encoding="utf-8") for path in public_files)
    for phrase in ("context pack", "release publish", "rc create", "verify release", "12 skills"):
        if phrase.lower() in public_text.lower():
            errors.append(f"removed command or surface remains in public docs: {phrase}")

    skill_names = sorted(path.name for path in (ROOT / "skills").iterdir() if (path / "SKILL.md").is_file())
    expected_skills = sorted(
        [
            "start-project",
            "continue-project",
            "plan-project-work",
            "coordinate-project-work",
            "recover-and-release-project",
        ]
    )
    if skill_names != expected_skills:
        errors.append(f"expected exactly five non-overlapping skills, found: {', '.join(skill_names)}")

    example_files = (
        ".agents/skills/use-project-harness/SKILL.md",
        ".agents/skills/use-project-harness/references/fallback.md",
        ".harness/entry.md",
        ".harness/manifest.yaml",
        ".harness/profile.yaml",
        ".harness/sources.yaml",
        ".harness/workspaces.yaml",
        ".harness/state/context-index.json",
        ".harness/state/impact-graph.json",
        ".harness/work/provider.yaml",
    )
    for example in ("starter", "multi-repo"):
        for relative in example_files:
            if not (ROOT / "examples" / example / relative).is_file():
                errors.append(f"example {example} misses generated harness file: {relative}")
    return errors


def main() -> int:
    errors = validate()
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Documentation parity passed: {len(PAIRS)} English/Korean pairs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
