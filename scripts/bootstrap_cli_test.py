from __future__ import annotations

import hashlib
import http.server
import os
import pathlib
import shlex
import shutil
import subprocess
import tempfile
import threading
import unittest

from render_plugin_packages import asset_name


ROOT = pathlib.Path(__file__).resolve().parents[1]


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, _format, *_args):
        return


class ReleaseServer:
    def __init__(self, directory: pathlib.Path):
        handler = lambda *args, **kwargs: QuietHandler(*args, directory=str(directory), **kwargs)
        self.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    def __enter__(self):
        self.thread.start()
        host, port = self.server.server_address
        return f"http://{host}:{port}/releases/download"

    def __exit__(self, *_args):
        self.server.shutdown()
        self.thread.join(timeout=5)
        self.server.server_close()


class BootstrapCLITest(unittest.TestCase):
    def setUp(self):
        self.shell = ROOT / "scripts/bootstrap-cli.sh"
        self.powershell = ROOT / "scripts/bootstrap-cli.ps1"

    def _fixture(self, root: pathlib.Path, checksum_override: str | None = None):
        release = root / "releases/download/v1.2.3"
        release.mkdir(parents=True)
        asset = asset_name("darwin", "arm64")
        payload = b'#!/bin/sh\n[ "$1" = doctor ] && printf \'{"status":"ok"}\\n\'\n'
        (release / asset).write_bytes(payload)
        (release / asset).chmod(0o755)
        digest = checksum_override or hashlib.sha256(payload).hexdigest()
        (release / "checksums.txt").write_text(f"{digest}  {asset}\n", encoding="utf-8")
        return asset

    def test_bootstrap_selects_exact_platform_asset_and_verifies_checksum(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            asset = self._fixture(root)
            install = root / "bin"
            with ReleaseServer(root) as base_url:
                completed = subprocess.run(
                    [
                        shutil.which("bash"), self.shell.as_posix(), "--base-url", base_url,
                        "--version", "1.2.3", "--install-dir", install.as_posix(),
                        "--os", "darwin", "--arch", "arm64",
                    ],
                    text=True,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    check=False,
                )
            self.assertEqual(0, completed.returncode, completed.stderr)
            self.assertTrue((install / "stackcord").is_file())
            self.assertIn('"status":"ok"', completed.stdout)
            self.assertEqual("stackcord_darwin_arm64", asset)

    def test_checksum_mismatch_never_installs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            self._fixture(root, checksum_override="0" * 64)
            install = root / "bin"
            with ReleaseServer(root) as base_url:
                completed = subprocess.run(
                    [
                        shutil.which("bash"), self.shell.as_posix(), "--base-url", base_url,
                        "--version", "1.2.3", "--install-dir", install.as_posix(),
                        "--os", "darwin", "--arch", "arm64",
                    ],
                    text=True,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    check=False,
                )
            self.assertNotEqual(0, completed.returncode)
            self.assertFalse((install / "stackcord").exists())

    def test_asset_matrix_is_complete(self):
        self.assertEqual("stackcord_darwin_amd64", asset_name("darwin", "amd64"))
        self.assertEqual("stackcord_darwin_arm64", asset_name("darwin", "arm64"))
        self.assertEqual("stackcord_windows_amd64.exe", asset_name("windows", "amd64"))
        self.assertEqual("stackcord_windows_arm64.exe", asset_name("windows", "arm64"))

    def test_powershell_is_checksum_first_and_atomic(self):
        text = self.powershell.read_text(encoding="utf-8")
        self.assertIn("Invoke-WebRequest", text)
        self.assertIn("Get-FileHash", text)
        self.assertIn("[System.IO.File]::Replace", text)
        self.assertIn("[System.IO.File]::Move", text)
        self.assertLess(text.index("Get-FileHash"), text.index("[System.IO.File]::Replace"))
        self.assertIn("doctor", text)
        for os_name, arch in (("windows", "amd64"), ("windows", "arm64")):
            self.assertIn(asset_name(os_name, arch), text)

    def test_session_hooks_never_download(self):
        hook_files = list((ROOT / "hooks").glob("*"))
        self.assertTrue((ROOT / "hooks/run-stackcord-hook.sh").is_file())
        self.assertTrue((ROOT / "hooks/run-stackcord-hook.ps1").is_file())
        hooks = "\n".join(path.read_text(encoding="utf-8") for path in hook_files if path.is_file())
        self.assertNotIn("curl", hooks)
        self.assertNotIn("Invoke-WebRequest", hooks)
        self.assertNotIn("bootstrap-cli", hooks)
        self.assertIn("PLUGIN_ROOT", hooks)

    def test_skills_offer_bootstrap_only_during_explicit_product_use(self):
        for name in ("start-project", "continue-project"):
            text = (ROOT / "skills" / name / "SKILL.md").read_text(encoding="utf-8")
            self.assertIn("STACKCORD_CLI", text)
            self.assertIn("bootstrap-cli", text)
            self.assertIn("explicit", text.lower())

    def test_hook_resolver_uses_explicit_cli_without_downloading(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            calls = root / "calls.txt"
            cli = root / "stackcord"
            cli.write_text(
                f"#!/bin/sh\nprintf '%s\\n' \"$*\" >> {shlex.quote(calls.as_posix())}\n",
                encoding="utf-8",
                newline="\n",
            )
            cli.chmod(0o755)
            env = dict(os.environ)
            env["STACKCORD_CLI"] = cli.as_posix()
            completed = subprocess.run(
                [shutil.which("bash"), (ROOT / "hooks/run-stackcord-hook.sh").as_posix(), "post-compact"],
                env=env,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )
            self.assertEqual(0, completed.returncode, completed.stderr)
            self.assertEqual("hook post-compact\n", calls.read_text(encoding="utf-8"))

    def test_both_hosts_resolve_same_cli_and_preserve_input_and_project_cwd(self):
        for variable in ("PLUGIN_ROOT", "CLAUDE_PLUGIN_ROOT"):
            with self.subTest(host=variable), tempfile.TemporaryDirectory(prefix="plugin space ") as directory:
                root = pathlib.Path(directory)
                (root / "bin").mkdir()
                cli = root / "bin/stackcord"
                cli.write_text('#!/bin/sh\nprintf "%s\\n" "$*"\ncat\n', encoding="utf-8", newline="\n")
                cli.chmod(0o755)
                env = {k:v for k,v in os.environ.items() if k not in ("PLUGIN_ROOT", "CLAUDE_PLUGIN_ROOT", "STACKCORD_CLI")}
                env[variable] = root.as_posix()
                result = subprocess.run([shutil.which("bash"), (ROOT / "hooks/run-stackcord-hook.sh").as_posix(), "session-start"],
                    cwd=ROOT, env=env, input='{"cwd":"project","source":"resume"}', text=True, capture_output=True)
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertEqual('hook session-start\n{"cwd":"project","source":"resume"}', result.stdout)

    def test_first_use_ui_offer_remembers_explicit_choice_across_hosts(self):
        for name in ("start-project", "continue-project"):
            text = (ROOT / "skills" / name / "SKILL.md").read_text(encoding="utf-8")
            for token in ("stackcord setup --ui enable --apply", "stackcord setup --ui disable --apply",
                          ".harness/local/control-center.json", "plugin cache", "stackcord dashboard"):
                self.assertIn(token, text)


if __name__ == "__main__":
    unittest.main()
