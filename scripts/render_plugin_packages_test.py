import hashlib
import json
import pathlib
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile

from render_plugin_packages import (
    asset_name,
    render_packages,
    stage_cli_assets,
    write_release_checksums,
)


ROOT = pathlib.Path(__file__).resolve().parents[1]


class RenderPluginPackagesTest(unittest.TestCase):
    def _verified_cli_assets(self, output):
        output.mkdir()
        lines = []
        for os_name, arch in (("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64"), ("windows", "arm64")):
            name = asset_name(os_name, arch)
            payload = f"verified current CLI {os_name}/{arch}".encode()
            (output / name).write_bytes(payload)
            lines.append(f"{hashlib.sha256(payload).hexdigest()}  {name}")
        (output / "checksums.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")

    def test_verified_cli_is_embedded_in_its_matching_platform_package(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            assets = root / "assets"
            self._verified_cli_assets(assets)
            packages = render_packages(ROOT, root / "packages", "1.0.0", "https://example.invalid/releases/download", cli_assets=assets)
            for package in packages:
                with zipfile.ZipFile(package) as archive:
                    platform = json.loads(archive.read("stackcord/distribution/platform.json"))
                    self.assertTrue(platform["bundledCLI"])
                    binary = "stackcord/bin/stackcord" + (".exe" if platform["os"] == "windows" else "")
                    self.assertEqual((assets / platform["asset"]).read_bytes(), archive.read(binary))
                    self.assertEqual(0o755, (archive.getinfo(binary).external_attr >> 16) & 0o777)
                    self.assertEqual([binary], [p for p in archive.namelist() if p.startswith("stackcord/bin/")])

    def test_embedding_requires_complete_checksum_verified_cli_set(self):
        for damage in ("missing", "mismatch"):
            with self.subTest(damage=damage), tempfile.TemporaryDirectory() as directory:
                root = pathlib.Path(directory)
                assets = root / "assets"
                self._verified_cli_assets(assets)
                target = assets / asset_name("windows", "arm64")
                if damage == "missing":
                    target.unlink()
                else:
                    target.write_bytes(b"corrupted")
                with self.assertRaises(ValueError):
                    render_packages(ROOT, root / "packages", "1.0.0", "https://example.invalid/releases/download", cli_assets=assets)
                self.assertFalse(list((root / "packages").glob("*.zip")))

    def test_embedding_rejects_symlinked_cli(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            assets = root / "assets"
            self._verified_cli_assets(assets)
            target = assets / asset_name("darwin", "arm64")
            linked = root / "external-cli"
            target.replace(linked)
            try:
                target.symlink_to(linked)
            except OSError as error:
                self.skipTest(f"host cannot create symlinks: {error}")
            with self.assertRaises(ValueError):
                render_packages(ROOT, root / "packages", "1.0.0", "https://example.invalid/releases/download", cli_assets=assets)

    def test_rendered_hook_uses_bundled_verified_cli_in_both_hosts(self):
        with tempfile.TemporaryDirectory(prefix="bundled plugin ") as directory:
            root = pathlib.Path(directory)
            assets = root / "assets"
            self._verified_cli_assets(assets)
            name = asset_name("darwin", "arm64")
            payload = b'#!/bin/sh\nprintf "%s\\n" "$*"\n'
            (assets / name).write_bytes(payload)
            entries = []
            for source in assets.iterdir():
                if source.name != "checksums.txt":
                    entries.append(f"{hashlib.sha256(source.read_bytes()).hexdigest()}  {source.name}")
            (assets / "checksums.txt").write_text("\n".join(entries) + "\n", encoding="utf-8")
            packages = render_packages(ROOT, root / "packages", "1.0.0", "https://example.invalid/releases/download", cli_assets=assets)
            with zipfile.ZipFile(packages[1]) as archive:
                archive.extractall(root / "unpacked")
            plugin = root / "unpacked/stackcord"
            (plugin / "bin/stackcord").chmod(0o755)
            for variable in ("PLUGIN_ROOT", "CLAUDE_PLUGIN_ROOT"):
                env = {k: v for k, v in os.environ.items() if k not in ("PLUGIN_ROOT", "CLAUDE_PLUGIN_ROOT", "STACKCORD_CLI")}
                env[variable] = plugin.as_posix()
                completed = subprocess.run([shutil.which("bash"), (plugin / "hooks/run-stackcord-hook.sh").as_posix(), "session-start"],
                                           env=env, capture_output=True, text=True, check=False)
                self.assertEqual(0, completed.returncode, completed.stderr)
                self.assertEqual("hook session-start\n", completed.stdout)

    def test_rendered_packages_bind_same_plugin_and_cli_version(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            packages = render_packages(
                root=ROOT,
                output=output,
                version="1.0.0",
                base_url="https://example.invalid/releases/download",
            )
            self.assertEqual(4, len(packages))
            for package in packages:
                with zipfile.ZipFile(package) as archive:
                    names = set(archive.namelist())
                    prefix = "stackcord/"
                    self.assertIn(prefix + ".codex-plugin/plugin.json", names)
                    self.assertIn(prefix + ".claude-plugin/plugin.json", names)
                    codex = json.loads(archive.read(prefix + ".codex-plugin/plugin.json"))
                    claude = json.loads(archive.read(prefix + ".claude-plugin/plugin.json"))
                    self.assertEqual(codex["version"], claude["version"])
                    self.assertEqual("./hooks/codex.json", codex["hooks"])
                    self.assertNotIn("hooks", claude)  # Claude discovers hooks/hooks.json once.
                    hooks = json.loads(archive.read(prefix + "hooks/hooks.json"))
                    for groups in hooks["hooks"].values():
                        for group in groups:
                            for hook in group["hooks"]:
                                self.assertIn("CLAUDE_PLUGIN_ROOT", hook["command"])
                                self.assertNotIn("commandWindows", hook)
                    self.assertIn(prefix + ".agents/plugins/marketplace.json", names)
                    self.assertIn(prefix + "distribution/platform.json", names)
                    self.assertIn(prefix + "skills/start-project/SKILL.md", names)
                    self.assertIn(prefix + "scripts/bootstrap-cli.sh", names)
                    self.assertNotIn(prefix + "AGENTS.md", names)
                    self.assertFalse(any(name.startswith(prefix + ".git/") for name in names))
                    self.assertFalse(any(name.startswith(prefix + ".harness/") for name in names))
                    self.assertFalse(any(name.endswith("_test.py") for name in names))
                    self.assertFalse(any(name.endswith((".go", "_test.go")) for name in names))
                    for excluded in ("evals/", "dogfood/", "testdata/", ".github/", "docs/superpowers/"):
                        self.assertFalse(any(name.startswith(prefix + excluded) for name in names))
                    platform = json.loads(archive.read(prefix + "distribution/platform.json"))
                    self.assertEqual("1.0.0", platform["pluginVersion"])
                    self.assertEqual("1.0.0", platform["cliVersion"])
                    self.assertEqual(
                        asset_name(platform["os"], platform["arch"]),
                        platform["asset"],
                    )

    def test_rendering_is_reproducible(self):
        with tempfile.TemporaryDirectory() as first, tempfile.TemporaryDirectory() as second:
            first_packages = render_packages(ROOT, pathlib.Path(first), "1.0.0", "https://example.invalid/releases/download")
            second_packages = render_packages(ROOT, pathlib.Path(second), "1.0.0", "https://example.invalid/releases/download")
            first_hashes = [hashlib.sha256(path.read_bytes()).hexdigest() for path in first_packages]
            second_hashes = [hashlib.sha256(path.read_bytes()).hexdigest() for path in second_packages]
            self.assertEqual(first_hashes, second_hashes)

    def test_rendered_package_validator_is_self_contained(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            package = render_packages(
                root=ROOT,
                output=root / "packages",
                version="1.0.0",
                base_url="https://example.invalid/releases/download",
            )[0]
            unpacked = root / "unpacked"
            with zipfile.ZipFile(package) as archive:
                archive.extractall(unpacked)
            plugin = unpacked / "stackcord"
            completed = subprocess.run(
                [sys.executable, str(plugin / "scripts" / "validate_plugin.py"), str(plugin)],
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )
            self.assertEqual(0, completed.returncode, completed.stderr)

    def test_goreleaser_binaries_are_staged_by_verified_upload_name(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            dist = root / "dist"
            output = dist / "release-assets"
            artifacts = []
            checksum_lines = []
            for os_name, arch in (("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64"), ("windows", "arm64")):
                name = asset_name(os_name, arch)
                source = dist / f"build-{os_name}-{arch}" / ("stackcord.exe" if os_name == "windows" else "stackcord")
                source.parent.mkdir(parents=True)
                source.write_bytes(f"{os_name}/{arch}".encode())
                digest = hashlib.sha256(source.read_bytes()).hexdigest()
                checksum_lines.append(f"{digest}  {name}")
                artifacts.append({
                    "name": name,
                    "path": str(source),
                    "goos": os_name,
                    "goarch": arch,
                    "type": "Binary",
                    "internal_type": 2,
                    "extra": {"Format": "binary", "Checksum": "sha256:" + digest},
                })
            (dist / "artifacts.json").write_text(json.dumps(artifacts), encoding="utf-8")
            (dist / "checksums.txt").write_text("\n".join(checksum_lines) + "\n", encoding="utf-8")
            staged = stage_cli_assets(dist, output)
            self.assertEqual(5, len(staged))
            self.assertEqual(
                {"checksums.txt"} | {asset_name(os_name, arch) for os_name, arch in (("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64"), ("windows", "arm64"))},
                {path.name for path in staged},
            )
            completed = subprocess.run([
                sys.executable, str(ROOT / "scripts/render_plugin_packages.py"),
                "--root", str(ROOT), "--output", str(output), "--version", "1.0.0",
                "--base-url", "https://example.invalid/releases/download", "--goreleaser-dist", str(dist),
            ], capture_output=True, text=True, check=False)
            self.assertEqual(0, completed.returncode, completed.stderr)
            self.assertEqual(4, len(list(output.glob("*.zip"))))
            for package in output.glob("*.zip"):
                with zipfile.ZipFile(package) as archive:
                    platform = json.loads(archive.read("stackcord/distribution/platform.json"))
                    self.assertTrue(platform["bundledCLI"])

    def test_release_checksums_cover_cli_and_plugin_assets(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            expected_names = {
                "stackcord_darwin_arm64",
                "stackcord_plugin_1.0.0_darwin_arm64.zip",
            }
            for name in expected_names:
                (output / name).write_bytes(name.encode())
            (output / "checksums.txt").write_text("stale\n", encoding="utf-8")

            manifest = write_release_checksums(output)

            entries = {}
            for line in manifest.read_text(encoding="utf-8").splitlines():
                digest, name = line.split("  ", 1)
                entries[name] = digest
            self.assertEqual(expected_names, set(entries))
            for name in expected_names:
                self.assertEqual(
                    hashlib.sha256((output / name).read_bytes()).hexdigest(),
                    entries[name],
                )



if __name__ == "__main__":
    unittest.main()
