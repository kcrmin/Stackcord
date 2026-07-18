import hashlib
import json
import pathlib
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
                    prefix = "fullstack-orchestrator/"
                    self.assertIn(prefix + ".codex-plugin/plugin.json", names)
                    self.assertIn(prefix + ".agents/plugins/marketplace.json", names)
                    self.assertIn(prefix + "distribution/platform.json", names)
                    self.assertIn(prefix + "skills/start-project/SKILL.md", names)
                    self.assertIn(prefix + "scripts/bootstrap-cli.sh", names)
                    self.assertFalse(any(name.startswith(prefix + ".git/") for name in names))
                    self.assertFalse(any(name.startswith(prefix + ".harness/") for name in names))
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
            plugin = unpacked / "fullstack-orchestrator"
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
                source = dist / f"build-{os_name}-{arch}" / ("orchestrator.exe" if os_name == "windows" else "orchestrator")
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

    def test_release_checksums_cover_cli_and_plugin_assets(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory)
            expected_names = {
                "orchestrator_darwin_arm64",
                "fullstack-orchestrator_plugin_1.0.0_darwin_arm64.zip",
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
