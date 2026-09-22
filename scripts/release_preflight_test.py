import json
import pathlib
import subprocess
import tempfile
import unittest

from release_preflight import validate_runs, validate_source


class ReleasePreflightTest(unittest.TestCase):
    def test_only_latest_successful_main_push_for_exact_source_qualifies(self):
        run = {"id": 10, "event": "push", "head_branch": "main",
               "head_sha": "a" * 40, "status": "completed", "conclusion": "success"}
        self.assertEqual([], validate_runs([run], "a" * 40))
        for change in ({"head_sha": "b" * 40}, {"event": "pull_request"},
                       {"head_branch": "feature/test"}, {"conclusion": "failure"},
                       {"conclusion": "cancelled"}, {"status": "in_progress"}):
            with self.subTest(change=change):
                self.assertTrue(validate_runs([dict(run, **change)], "a" * 40))
        self.assertTrue(validate_runs([], "a" * 40))
        self.assertTrue(validate_runs([dict(run, id=11, conclusion="failure"), run], "a" * 40))

    def test_tag_must_match_expected_source_main_history_and_manifest_version(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            def git(*args):
                return subprocess.check_output(["git", *args], cwd=root, text=True).strip()
            git("init", "-q", "-b", "main")
            git("config", "user.name", "Release test")
            git("config", "user.email", "release-test@example.invalid")
            (root / ".codex-plugin").mkdir()
            (root / ".codex-plugin/plugin.json").write_text(json.dumps({"version": "1.2.3"}))
            git("add", ".")
            git("commit", "-qm", "test fixture")
            sha = git("rev-parse", "HEAD")
            git("update-ref", "refs/remotes/origin/main", sha)
            git("tag", "-a", "v1.2.3", "-m", "candidate")
            self.assertEqual([], validate_source(root, "v1.2.3", sha))
            self.assertTrue(validate_source(root, "v1.2.3", "b" * 40))
            self.assertTrue(validate_source(root, "--help", sha))
            self.assertTrue(validate_source(root, "v1.2.3", "HEAD"))
            git("tag", "v9.9.9")
            self.assertTrue(validate_source(root, "v9.9.9", sha))
            git("checkout", "-qb", "feature/test")
            git("commit", "--allow-empty", "-qm", "unmerged source")
            unmerged = git("rev-parse", "HEAD")
            git("tag", "v1.2.4")
            self.assertTrue(any("main" in e for e in validate_source(root, "v1.2.4", unmerged)))


if __name__ == "__main__":
    unittest.main()
