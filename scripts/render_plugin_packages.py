#!/usr/bin/env python3
"""Render reproducible platform-specific Plugin zip packages."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import stat
import zipfile


PLATFORMS = (("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64"), ("windows", "arm64"))
SEMVER = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?$")
TOP_LEVEL_FILES = (
    "AGENTS.md",
    "LICENSE",
    "README.md",
    "README.ko.md",
    ".agents/plugins/marketplace.json",
)
PACKAGE_TREES = (".codex-plugin", "hooks", "skills", "references", "templates", "schemas", "profiles/strict-release")
PACKAGE_SCRIPTS = ("scripts/bootstrap-cli.sh", "scripts/bootstrap-cli.ps1", "scripts/validate_plugin.py")
ZIP_TIME = (1980, 1, 1, 0, 0, 0)


def asset_name(os_name: str, arch: str) -> str:
    if (os_name, arch) not in PLATFORMS:
        raise ValueError(f"unsupported platform: {os_name}/{arch}")
    suffix = ".exe" if os_name == "windows" else ""
    return f"orchestrator_{os_name}_{arch}{suffix}"


def _plugin_manifest(root: pathlib.Path) -> dict[str, object]:
    return json.loads((root / ".codex-plugin/plugin.json").read_text(encoding="utf-8"))


def _package_files(root: pathlib.Path) -> list[pathlib.Path]:
    files: list[pathlib.Path] = []
    for relative in TOP_LEVEL_FILES + PACKAGE_SCRIPTS:
        path = root / relative
        if not path.is_file() or path.is_symlink():
            raise ValueError(f"missing or unsafe package file: {relative}")
        files.append(path)
    for relative in PACKAGE_TREES:
        tree = root / relative
        if not tree.is_dir() or tree.is_symlink():
            raise ValueError(f"missing or unsafe package tree: {relative}")
        for path in tree.rglob("*"):
            if path.is_symlink():
                raise ValueError(f"package tree contains symlink: {path.relative_to(root)}")
            if path.is_file():
                files.append(path)
    return sorted(set(files), key=lambda path: path.relative_to(root).as_posix())


def _write_entry(archive: zipfile.ZipFile, name: str, data: bytes, executable: bool = False) -> None:
    info = zipfile.ZipInfo(name, ZIP_TIME)
    info.compress_type = zipfile.ZIP_DEFLATED
    mode = 0o755 if executable else 0o644
    info.external_attr = (stat.S_IFREG | mode) << 16
    info.create_system = 3
    archive.writestr(info, data)


def render_packages(
    root: pathlib.Path,
    output: pathlib.Path,
    version: str,
    base_url: str,
) -> list[pathlib.Path]:
    root = root.resolve()
    output = output.resolve()
    if not SEMVER.fullmatch(version):
        raise ValueError(f"invalid version: {version}")
    if not base_url.startswith("https://"):
        raise ValueError("release base URL must use HTTPS")
    manifest = _plugin_manifest(root)
    if manifest.get("version") != version:
        raise ValueError(
            f"Plugin version {manifest.get('version')!r} does not match CLI package version {version!r}"
        )
    plugin_name = str(manifest["name"])
    sources = _package_files(root)
    output.mkdir(parents=True, exist_ok=True)
    rendered: list[pathlib.Path] = []
    for os_name, arch in PLATFORMS:
        destination = output / f"{plugin_name}_plugin_{version}_{os_name}_{arch}.zip"
        platform = {
            "schemaVersion": 1,
            "pluginVersion": version,
            "cliVersion": version,
            "os": os_name,
            "arch": arch,
            "asset": asset_name(os_name, arch),
            "checksums": f"{base_url.rstrip('/')}/v{version}/checksums.txt",
            "bootstrap": "scripts/bootstrap-cli.ps1" if os_name == "windows" else "scripts/bootstrap-cli.sh",
        }
        with zipfile.ZipFile(destination, "w") as archive:
            prefix = f"{plugin_name}/"
            for source in sources:
                relative = source.relative_to(root).as_posix()
                executable = relative == "scripts/bootstrap-cli.sh"
                _write_entry(archive, prefix + relative, source.read_bytes(), executable)
            _write_entry(
                archive,
                prefix + "distribution/platform.json",
                (json.dumps(platform, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode("utf-8"),
            )
        rendered.append(destination)
    return rendered


def _checksum_map(path: pathlib.Path) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        parts = line.split(maxsplit=1)
        if len(parts) != 2:
            continue
        digest, name = parts[0].lower(), parts[1].lstrip("*")
        if not re.fullmatch(r"[0-9a-f]{64}", digest) or name in checksums:
            raise ValueError("checksum manifest contains an invalid or duplicate entry")
        checksums[name] = digest
    return checksums


def _atomic_copy(source: pathlib.Path, destination: pathlib.Path) -> None:
    temporary = destination.with_name(f".{destination.name}.tmp-{os.getpid()}")
    try:
        shutil.copyfile(source, temporary)
        os.replace(temporary, destination)
    finally:
        if temporary.exists():
            temporary.unlink()


def stage_cli_assets(dist: pathlib.Path, output: pathlib.Path) -> list[pathlib.Path]:
    dist = dist.resolve()
    output = output.resolve()
    artifacts_path = dist / "artifacts.json"
    checksums_path = dist / "checksums.txt"
    if not artifacts_path.is_file() or not checksums_path.is_file():
        raise ValueError("GoReleaser artifacts.json or checksums.txt is absent")
    artifacts = json.loads(artifacts_path.read_text(encoding="utf-8"))
    if not isinstance(artifacts, list):
        raise ValueError("GoReleaser artifacts.json must be a list")
    expected_names = {asset_name(os_name, arch) for os_name, arch in PLATFORMS}
    selected: dict[str, pathlib.Path] = {}
    declared_digests: dict[str, str] = {}
    for artifact in artifacts:
        if not isinstance(artifact, dict) or artifact.get("type") != "Binary":
            continue
        extra = artifact.get("extra", {})
        if artifact.get("internal_type") != 2 or not isinstance(extra, dict) or extra.get("Format") != "binary":
            continue
        os_name, arch = artifact.get("goos"), artifact.get("goarch")
        try:
            expected = asset_name(str(os_name), str(arch))
        except ValueError as error:
            raise ValueError(f"unexpected GoReleaser binary platform: {os_name}/{arch}") from error
        name = str(artifact.get("name", ""))
        if name != expected or name in selected:
            raise ValueError(f"unexpected or duplicate GoReleaser upload name: {name}")
        source = pathlib.Path(str(artifact.get("path", "")))
        if not source.is_absolute():
            source = dist.parent / source
        source = source.resolve()
        try:
            source.relative_to(dist)
        except ValueError as error:
            raise ValueError(f"GoReleaser artifact escapes dist: {source}") from error
        if not source.is_file() or source.is_symlink():
            raise ValueError(f"GoReleaser artifact is missing or unsafe: {source}")
        declared = str(extra.get("Checksum", ""))
        if not declared.startswith("sha256:"):
            raise ValueError(f"GoReleaser artifact lacks SHA-256: {name}")
        selected[name] = source
        declared_digests[name] = declared.removeprefix("sha256:").lower()
    if set(selected) != expected_names:
        raise ValueError(f"GoReleaser platform assets differ: {sorted(selected)}")

    checksums = _checksum_map(checksums_path)
    output.mkdir(parents=True, exist_ok=True)
    staged: list[pathlib.Path] = []
    for name in sorted(selected):
        digest = hashlib.sha256(selected[name].read_bytes()).hexdigest()
        if digest != declared_digests[name] or digest != checksums.get(name):
            raise ValueError(f"GoReleaser checksum mismatch: {name}")
        destination = output / name
        _atomic_copy(selected[name], destination)
        if not name.endswith(".exe"):
            destination.chmod(0o755)
        staged.append(destination)
    checksum_destination = output / "checksums.txt"
    _atomic_copy(checksums_path, checksum_destination)
    staged.append(checksum_destination)
    return staged


def write_release_checksums(output: pathlib.Path) -> pathlib.Path:
    """Write one deterministic checksum manifest for every staged release asset."""
    output = output.resolve()
    if not output.is_dir() or output.is_symlink():
        raise ValueError("release asset directory is missing or unsafe")
    assets = []
    for path in output.iterdir():
        if path.name == "checksums.txt":
            continue
        if not path.is_file() or path.is_symlink():
            raise ValueError(f"release output contains a non-file or symlink: {path.name}")
        assets.append(path)
    if not assets:
        raise ValueError("release output contains no assets")
    lines = [
        f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}"
        for path in sorted(assets, key=lambda item: item.name)
    ]
    destination = output / "checksums.txt"
    temporary = destination.with_name(f".{destination.name}.tmp-{os.getpid()}")
    try:
        temporary.write_text("\n".join(lines) + "\n", encoding="utf-8")
        os.replace(temporary, destination)
    finally:
        if temporary.exists():
            temporary.unlink()
    return destination


def parser() -> argparse.ArgumentParser:
    value = argparse.ArgumentParser(description=__doc__)
    value.add_argument("--root", default=".")
    value.add_argument("--output", required=True)
    value.add_argument("--version", required=True)
    value.add_argument("--base-url", required=True)
    value.add_argument("--goreleaser-dist")
    return value


def main() -> int:
    args = parser().parse_args()
    packages = render_packages(
        pathlib.Path(args.root), pathlib.Path(args.output), args.version, args.base_url
    )
    staged = []
    if args.goreleaser_dist:
        staged = stage_cli_assets(pathlib.Path(args.goreleaser_dist), pathlib.Path(args.output))
    manifest = write_release_checksums(pathlib.Path(args.output))
    assets = packages + [path for path in staged if path.name != "checksums.txt"] + [manifest]
    for package in assets:
        print(package)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
