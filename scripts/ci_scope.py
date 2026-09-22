"""Conservative CI scope and fail-closed stable gate, using no third-party API."""
import argparse
import json
import os
import subprocess


def security_gate_passes(event, states):
    if event not in ("pull_request", "push", "schedule", "workflow_dispatch"):
        return False
    expected = {"codeql": "success", "vulnerability": "success",
                "dependency-review": "success" if event == "pull_request" else "skipped",
                "scheduled-concurrency-and-fuzz": "success" if event in ("schedule", "workflow_dispatch") else "skipped"}
    return all(states.get(job) == result for job, result in expected.items())


def needs_full_tests(paths):
    return not paths or any(not (p in ("README.md", "README.ko.md") or
        (p.startswith("docs/") and p.endswith(".md"))) for p in paths)


def gate_passes(full, states):
    if any(states.get(job) != "success" for job in ("changes", "repository-contracts")):
        return False
    expected = "success" if full else "skipped"
    return all(states.get(job) == expected for job in ("native", "product-dogfood", "cross-build"))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base")
    parser.add_argument("--head", default="HEAD")
    parser.add_argument("--gate", action="store_true")
    parser.add_argument("--security-gate", action="store_true")
    args = parser.parse_args()
    if args.gate or args.security_gate:
        needs = json.loads(os.environ["CI_NEEDS"])
        states = {k: v.get("result") for k, v in needs.items()}
        choice = needs.get("changes", {}).get("outputs", {}).get("full")
        passed = (security_gate_passes(os.environ["GITHUB_EVENT_NAME"], states)
                  if args.security_gate else choice in ("true", "false") and gate_passes(choice == "true", states))
        report = "| Check | Result |\n| --- | --- |\n" + "".join(f"| {k} | {v} |\n" for k, v in states.items())
        print(report)
        if os.environ.get("GITHUB_STEP_SUMMARY"):
            with open(os.environ["GITHUB_STEP_SUMMARY"], "a", encoding="utf-8") as summary:
                summary.write(report)
        if not passed:
            print("Required checks failed, were cancelled, or were unexpectedly skipped.")
        return 0 if passed else 1
    full = True
    if args.base and set(args.base) != {"0"}:
        diff = subprocess.run(["git", "diff", "--name-only", "-z", args.base, args.head, "--"],
                              capture_output=True, check=False)
        if diff.returncode == 0:
            full = needs_full_tests([p.decode("utf-8", errors="replace") for p in diff.stdout.split(b"\0") if p])
    line = "full=" + str(full).lower()
    print(line)
    with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
        output.write(line + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
