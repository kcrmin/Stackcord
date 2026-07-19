#!/usr/bin/env python3
"""Check English/Korean document pairing and catalog parity."""

import json
import pathlib
import re
import shlex
import subprocess
import sys
import tempfile


ROOT = pathlib.Path(__file__).resolve().parents[1]
PAIRS = [
    ("docs/getting-started/en.md", "docs/getting-started/ko.md"),
    ("docs/concepts/en.md", "docs/concepts/ko.md"),
    *[(f"docs/guides/{name}-en.md", f"docs/guides/{name}-ko.md") for name in ("new-project", "existing-project", "submodules", "task-management", "governance", "dbdiagram", "ui-workspace", "release", "troubleshooting")],
    *[(f"docs/security/{name}-en.md", f"docs/security/{name}-ko.md") for name in ("threat-model", "privacy")],
]
SKILL_NAMES = (
    "start-project",
    "continue-project",
    "plan-project-work",
    "coordinate-project-work",
    "recover-and-release-project",
)


def headings(text: str) -> list[int]:
    return [len(match.group(1)) for match in re.finditer(r"^(#+) ", text, re.MULTILINE)]


def extract_orchestrator_commands(text: str) -> set[tuple[str, ...]]:
    commands: set[tuple[str, ...]] = set()
    for match in re.finditer(r"(?<![A-Za-z0-9_-])orchestrator(?:[ \t]+[^\s`]+)*", text):
        try:
            tokens = shlex.split(match.group(0))
        except ValueError:
            continue
        path: list[str] = []
        for token in tokens[1:]:
            token = token.rstrip(".,;:)")
            if token.startswith("-") or not re.fullmatch(r"[a-z][a-z0-9-]*", token):
                break
            path.append(token)
        if path:
            commands.add(tuple(path))
    return commands


def public_contract_errors(documents: dict[str, str]) -> list[str]:
    errors: list[str] = []
    for path in ("README.md", "README.ko.md"):
        text = documents.get(path, "")
        if not all(name in text for name in SKILL_NAMES):
            errors.append(f"{path} must name the same five Skills")
        if not all(token in text for token in ("feature/account-recovery", "feat(account):", "AI")):
            errors.append(f"{path} must describe AI-free Git conventions")
        if not all(path_token in text for path_token in (
            ".agents/skills/use-project-harness/", ".harness/work/provider.yaml", ".harness/governance.yaml", "contracts/registry.yaml",
        )):
            errors.append(f"{path} generated project paths differ from the tested fixture")
        if ".harness/local/context/" not in text or ".harness/state/context-index.json" in text:
            errors.append(f"{path} must describe the ignored generated context location")
        if not all(token in text for token in ("```mermaid", "ui/", "frontend/", "strict", "ui-workspace")):
            errors.append(f"{path} must show the concise UI-to-release product flow")

    reader_contracts = {
        "README.md": {
            "positioning": ("Stackcord", "Question-Driven Development", "full-stack", "release"),
            "conversation": ("A.", "Recommended", "free-form"),
            "external": ("GitHub Issues + Git", "Beads + Git", "Superpowers", "BMAD"),
            "install": ("Install the Stackcord Plugin from this GitHub link", "codex plugin marketplace add"),
        },
        "README.ko.md": {
            "positioning": ("Stackcord", "Question-Driven Development", "풀스택", "release"),
            "conversation": ("A.", "추천", "직접 입력"),
            "external": ("GitHub Issues + Git", "Beads + Git", "Superpowers", "BMAD"),
            "install": ("이 GitHub 링크의 Stackcord Plugin을 설치", "codex plugin marketplace add"),
        },
    }
    for path, contract in reader_contracts.items():
        text = documents.get(path, "")
        if not all(token in text for token in contract["positioning"]):
            errors.append(f"{path} must present the approved Stackcord positioning")
        if not all(token in text for token in contract["conversation"]):
            errors.append(f"{path} must show a recommended choice conversation with free-form input")
        if not all(token in text for token in contract["external"]):
            errors.append(f"{path} must show an external tool recommendation at the point of need")
        if not all(token in text for token in contract["install"]):
            errors.append(f"{path} must lead with natural-language installation and keep CLI as fallback")
        if "Go 1.26" in text or "go build" in text:
            errors.append(f"{path} must not present Go source builds as an end-user prerequisite")

    governance_requirements = {
        "README.md": ("People and AI understand the service differently", "product authorities", "governance-en.md"),
        "README.ko.md": ("사람과 AI마다 서비스의 목적·정책·동작을 다르게 이해", "제품 책임자", "governance-ko.md"),
    }
    for path, required in governance_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must explain shared product meaning and product authority")

    concept_requirements = {
        "docs/concepts/en.md": ("Memory is not", "repository evidence", "canonical"),
        "docs/concepts/ko.md": ("Memory는", "저장소 evidence", "canonical"),
    }
    for path, required in concept_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must distinguish Memory from canonical repository evidence")

    provider_requirements = {
        "docs/guides/task-management-en.md": ("one live status source", "Git-local", "GitHub", "Jira", "Beads", "cached"),
        "docs/guides/task-management-ko.md": ("live status 원본 하나", "Git-local", "GitHub", "Jira", "Beads", "cache"),
    }
    for path, required in provider_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must explain the provider truth boundary")

    return errors


def safety_contract_errors(documents: dict[str, str]) -> list[str]:
    errors: list[str] = []
    threat_requirements = {
        "docs/security/threat-model-en.md": (
            "prompt injection", "normalized observation", "compare-and-swap", "symlink",
            "path traversal", "archive size", "submodule URL", "allowlist", "exact candidate",
            "hosted provider",
        ),
        "docs/security/threat-model-ko.md": (
            "prompt injection", "normalized observation", "compare-and-swap", "symlink",
            "path traversal", "archive size", "submodule URL", "allowlist", "exact candidate",
            "hosted provider",
        ),
    }
    for path, required in threat_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must explain the complete threat boundaries")

    privacy_requirements = {
        "docs/security/privacy-en.md": ("raw provider payload", "local observation", "not committed", "retention"),
        "docs/security/privacy-ko.md": ("provider 원본 payload", "local observation", "commit하지", "retention"),
    }
    for path, required in privacy_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must explain provider observation privacy")

    outage_requirements = {
        "docs/guides/troubleshooting-en.md": ("remains selected", "unknown", "explicitly switch"),
        "docs/guides/troubleshooting-ko.md": ("선택 상태를 유지", "unknown", "명시적으로 전환"),
    }
    for path, required in outage_requirements.items():
        text = documents.get(path, "")
        if not all(token in text for token in required):
            errors.append(f"{path} must not silently change providers during a provider outage")

    return errors


def verify_documented_commands(commands: set[tuple[str, ...]]) -> list[str]:
    if not commands:
        return ["public documentation contains no executable orchestrator commands"]
    with tempfile.TemporaryDirectory(prefix="orchestrator-docs-") as directory:
        executable = pathlib.Path(directory) / ("orchestrator.exe" if sys.platform == "win32" else "orchestrator")
        build = subprocess.run(
            ["go", "build", "-trimpath", "-o", str(executable), "./cmd/orchestrator"],
            cwd=ROOT / "cli",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if build.returncode != 0:
            return [f"cannot build CLI for documentation validation: {build.stderr.strip()}"]
        errors: list[str] = []
        for path in sorted(commands):
            result = subprocess.run(
                [str(executable), *path, "--help"],
                cwd=ROOT,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )
            if result.returncode != 0:
                errors.append(f"documented CLI command does not exist: orchestrator {' '.join(path)}")
        return errors


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
    expected_skills = sorted(SKILL_NAMES)
    if skill_names != expected_skills:
        errors.append(f"expected exactly five non-overlapping skills, found: {', '.join(skill_names)}")

    example_files = (
        ".agents/skills/use-project-harness/SKILL.md",
        ".agents/skills/use-project-harness/references/fallback.md",
        ".harness/entry.md",
        ".harness/manifest.yaml",
        ".harness/profile.yaml",
        ".harness/governance.yaml",
        ".harness/sources.yaml",
        ".harness/workspaces.yaml",
        ".harness/work/provider.yaml",
    )
    for example in ("starter", "multi-repo"):
        for relative in example_files:
            if not (ROOT / "examples" / example / relative).is_file():
                errors.append(f"example {example} misses generated harness file: {relative}")
        for obsolete in (".harness/state/context-index.json", ".harness/state/impact-graph.json"):
            if (ROOT / "examples" / example / obsolete).exists():
                errors.append(f"example {example} tracks generated context cache: {obsolete}")
    required_documents = {
        path: (ROOT / path).read_text(encoding="utf-8") if (ROOT / path).is_file() else ""
        for path in (
            "README.md", "README.ko.md", "docs/concepts/en.md", "docs/concepts/ko.md",
            "docs/guides/task-management-en.md", "docs/guides/task-management-ko.md",
            "docs/guides/troubleshooting-en.md", "docs/guides/troubleshooting-ko.md",
            "docs/security/threat-model-en.md", "docs/security/threat-model-ko.md",
            "docs/security/privacy-en.md", "docs/security/privacy-ko.md",
        )
    }
    errors.extend(public_contract_errors(required_documents))
    errors.extend(safety_contract_errors(required_documents))
    errors.extend(verify_documented_commands(extract_orchestrator_commands(public_text)))
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
