"""Verify the source and completed main-branch checks before release staging."""
import argparse
import json
import os
import pathlib
import re
import subprocess
import sys


def validate_runs(runs, sha):
    matching = [run for run in runs if run.get("head_sha") == sha
                and run.get("head_branch") == "main" and run.get("event") == "push"]
    if not matching:
        return ["no main push run for the exact source commit"]
    latest = max(matching, key=lambda run: run["id"])
    if latest.get("status") != "completed" or latest.get("conclusion") != "success":
        return [f"latest run {latest['id']} is {latest.get('status')}/{latest.get('conclusion')}"]
    return []


def validate_source(root, tag, sha):
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", tag):
        return ["tag must be an exact vMAJOR.MINOR.PATCH release tag"]
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        return ["expected source must be a full lowercase commit SHA"]
    def git(*args):
        return subprocess.check_output(["git", *args], cwd=root, text=True, stderr=subprocess.PIPE).strip()
    try:
        if git("rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}") != sha:
            return ["release tag does not match the expected source commit"]
        git("merge-base", "--is-ancestor", sha, "refs/remotes/origin/main")
    except subprocess.CalledProcessError:
        return ["release tag is missing or its source is not in origin/main"]
    try:
        manifest = json.loads(git("show", f"{sha}:.codex-plugin/plugin.json"))
        if manifest.get("version") != tag[1:]:
            return ["release tag version differs from the source Plugin manifest"]
    except (subprocess.CalledProcessError, ValueError):
        return ["release source has no valid Plugin manifest"]
    return []


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--expected-sha", required=True)
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY"))
    args = parser.parse_args()
    errors = validate_source(pathlib.Path.cwd(), args.tag, args.expected_sha)
    if not errors:
        if not args.repo or not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", args.repo):
            errors.append("a GitHub owner/repository is required")
        else:
            for workflow in ("ci.yml", "security.yml"):
                endpoint = (f"repos/{args.repo}/actions/workflows/{workflow}/runs"
                            f"?head_sha={args.expected_sha}&branch=main&event=push&per_page=100")
                try:
                    pages = json.loads(subprocess.check_output(
                        ["gh", "api", "--paginate", "--slurp", endpoint], text=True))
                    runs = [run for page in pages for run in page["workflow_runs"]]
                    errors.extend(f"{workflow}: {error}" for error in validate_runs(runs, args.expected_sha))
                except (subprocess.CalledProcessError, ValueError, KeyError, TypeError):
                    errors.append(f"{workflow}: could not verify GitHub Actions evidence")
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Verified {args.tag} at {args.expected_sha}: CI and Security passed on main")
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            output.write(f"source_sha={args.expected_sha}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
