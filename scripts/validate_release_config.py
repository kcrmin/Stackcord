#!/usr/bin/env python3
"""Validate CI, supply-chain, packaging, and publish fail-closed invariants."""

import pathlib
import re
import sys


SHA_PIN = re.compile(r"^\s*-?\s*uses:\s*[^@\s]+@([0-9a-f]{40})\s*$", re.MULTILINE)
USES = re.compile(r"^\s*-?\s*uses:\s*([^\s]+)\s*$", re.MULTILINE)


def validate(root: pathlib.Path) -> list[str]:
    errors: list[str] = []
    required = [
        ".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/workflows/release.yml",
        ".goreleaser.yaml", "scripts/bootstrap-cli.sh", "scripts/bootstrap-cli.ps1",
        "scripts/render_plugin_packages.py", "profiles/strict-release/README.md",
        "profiles/strict-release/packaging/homebrew/orchestrator.rb",
        "profiles/strict-release/packaging/winget/FullstackOrchestrator.Orchestrator.installer.yaml",
        "profiles/strict-release/packaging/windows/Product.wxs",
        "profiles/strict-release/scripts/generate_packages.py",
        "profiles/strict-release/scripts/verify_publish_guard.py",
        "profiles/strict-release/scripts/verify_staged_release.py",
        "SECURITY.md", "SUPPORT.md", "CONTRIBUTING.md", "GOVERNANCE.md", "LICENSE",
    ]
    for relative in required:
        if not (root / relative).is_file():
            errors.append(f"required release file missing: {relative}")

    workflows = []
    for path in sorted((root / ".github" / "workflows").glob("*.yml")):
        text = path.read_text(encoding="utf-8")
        workflows.append(text)
        for use in USES.findall(text):
            if not re.search(r"@[0-9a-f]{40}$", use):
                errors.append(f"GitHub Action is not commit-pinned: {path.name}: {use}")
    ci = (root / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8") if (root / ".github" / "workflows" / "ci.yml").exists() else ""
    for label in ("macos-14", "windows-2025"):
        if label not in ci:
            errors.append(f"representative native CI target missing: {label}")
    for label in ("macos-15-intel", "windows-11-arm"):
        if label in ci:
            errors.append(f"non-representative target repeats the full PR suite: {label}")
    for target in ("darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"):
        if target not in ci:
            errors.append(f"cross-build target missing: {target}")
    if "validate_plugin.py" not in ci:
        errors.append("CI lacks Plugin evidence")

    security = (root / ".github" / "workflows" / "security.yml").read_text(encoding="utf-8") if (root / ".github" / "workflows" / "security.yml").exists() else ""
    for evidence in ("govulncheck", "dependency-review-action", "codeql-action"):
        if evidence not in security:
            errors.append(f"security workflow missing {evidence}")
    for evidence in ("go test -race ./...", "-fuzz FuzzFingerprint"):
        if evidence not in security:
            errors.append(f"scheduled security workflow missing {evidence}")
    if "security_scan.py" not in ci:
        errors.append("pull-request contracts lack the repository secret scan")
    if "security_scan.py" in security:
        errors.append("repository secret scan is duplicated across pull-request workflows")

    release = (root / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8") if (root / ".github" / "workflows" / "release.yml").exists() else ""
    for guard in ("workflow_dispatch", "environment: production", "rc_digest", "--skip=publish", "render_plugin_packages.py", "checksums.txt", "--draft", "gh release create"):
        if guard not in release:
            errors.append(f"release workflow missing fail-closed guard: {guard}")
    for strict_token in ("approval_operation_id", "verify_publish_guard.py", "cosign", "sigstore"):
        if strict_token in release:
            errors.append(f"default release workflow contains strict-only control: {strict_token}")
    if "pull_request" in release.split("jobs:", 1)[0]:
        errors.append("release workflow must not publish from pull requests")
    for name, workflow in (("CI", ci), ("security", security), ("release", release)):
        if "run_agent_eval.py" in workflow or "codex exec" in workflow:
            errors.append(f"{name} workflow must not execute model evaluations")

    config = (root / ".goreleaser.yaml").read_text(encoding="utf-8") if (root / ".goreleaser.yaml").exists() else ""
    for token in ("CGO_ENABLED=0", "darwin", "windows", "amd64", "arm64", "-trimpath", "formats: [binary]", "orchestrator_{{ .Os }}_{{ .Arch }}", "checksums.txt"):
        if token not in config:
            errors.append(f"GoReleaser configuration missing {token}")
    for strict_token in ("sboms:", "signs:", "cosign"):
        if strict_token in config:
            errors.append(f"default GoReleaser contains strict-only control: {strict_token}")

    package_root = root / "profiles" / "strict-release" / "packaging"
    package_text = "\n".join(path.read_text(encoding="utf-8") for path in package_root.rglob("*") if path.is_file()) if package_root.exists() else ""
    for token in ("DARWIN_ARM64_SHA256", "DARWIN_AMD64_SHA256", "WINDOWS_ARM64_SHA256", "WINDOWS_AMD64_SHA256"):
        if token not in package_text:
            errors.append(f"package checksum token missing: {token}")
    license_text = (root / "LICENSE").read_text(encoding="utf-8") if (root / "LICENSE").exists() else ""
    if "Apache License" not in license_text or "Version 2.0" not in license_text:
        errors.append("Apache-2.0 license text is incomplete")
    return errors


def main() -> int:
    root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = validate(root)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print("Release configuration validation passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
