#!/usr/bin/env python3
"""Exercise the product against real temporary Git repositories and submodules."""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import textwrap
from dataclasses import dataclass
from typing import Any


SCENARIO = "scenario.multi-repository-continuity"
ROOT_URL = "https://example.test/orchestration.git"
FRONTEND_URL = "https://example.test/frontend.git"
BACKEND_URL = "https://example.test/backend.git"
PARENT_WORK = "work.account-recovery"
BACKEND_WORK = "work.account-recovery-backend"
FRONTEND_WORK = "work.account-recovery-frontend"
CONTRACT_ID = "contract.account-recovery.v1"


class ScenarioError(RuntimeError):
    pass


@dataclass
class CommandResult:
    returncode: int
    stdout: str
    stderr: str


class Dogfood:
    def __init__(self, binary: pathlib.Path, workspace: pathlib.Path, output: pathlib.Path) -> None:
        self.binary = binary.resolve()
        self.workspace = workspace.resolve()
        self.output = output.resolve()
        self.product_root = pathlib.Path(__file__).resolve().parents[1]
        self.assertions: list[dict[str, str]] = []
        self.tdd_failures = 0
        self.cli_calls = 0
        self.git_calls = 0
        self.remotes = {
            ROOT_URL: self.workspace / "remotes" / "orchestration.git",
            FRONTEND_URL: self.workspace / "remotes" / "frontend.git",
            BACKEND_URL: self.workspace / "remotes" / "backend.git",
        }

    def run(self) -> None:
        if not self.binary.is_file():
            raise ScenarioError("CLI binary is absent")
        if self.workspace.exists() and any(self.workspace.iterdir()):
            raise ScenarioError("dogfood workspace must be absent or empty")
        self.workspace.mkdir(parents=True, exist_ok=True)
        (self.workspace / "remotes").mkdir()
        for remote in self.remotes.values():
            self.git(self.workspace, "init", "--bare", "--initial-branch=main", str(remote))

        self.seed_child("frontend", FRONTEND_URL)
        self.seed_child("backend", BACKEND_URL)
        root = self.initialize_root()
        self.prove_claim_race_and_conflict(root)
        self.start_child_work(root)
        backend_evidence = self.implement_backend(root)
        frontend_evidence = self.implement_frontend(root)
        self.require(
            "evidence.backend-provider",
            backend_evidence,
            "backend failing test, passing test, and commit-bound evidence were observed",
        )
        self.require(
            "evidence.frontend-consumer",
            frontend_evidence,
            "frontend failing test, passing test, and commit-bound evidence were observed",
        )
        self.integrate_child_pointers(root)
        self.prove_parent_worktree_and_evidence(root)
        self.prove_integration(root)
        self.prove_release(root)
        self.prove_clone_recovery(root)
        self.prove_product_self_adoption()

    def prove_product_self_adoption(self) -> None:
        source = self.product_root
        target = self.workspace / "product-self-adopt"

        def ignore(directory: str, names: list[str]) -> set[str]:
            ignored = {name for name in names if name in {".git", ".worktrees", ".tools", "dist", "__pycache__"} or name.endswith(".pyc")}
            if pathlib.Path(directory).resolve() == source and ".harness" in names:
                ignored.add(".harness")
            return ignored

        shutil.copytree(source, target, ignore=ignore)
        protected = [pathlib.Path("cli/go.mod"), pathlib.Path("skills/start-project/SKILL.md")]
        before = {str(path): hashlib.sha256((target / path).read_bytes()).hexdigest() for path in protected}
        adopted = self.cli(
            "project", "adopt", "--root", str(target), "--id", "project.service-continuity-product",
            "--name", "Service Continuity Product", "--locale", "en", "--apply",
        )
        self.expect_status(adopted, "passed", "product self-adoption")
        after = {str(path): hashlib.sha256((target / path).read_bytes()).hexdigest() for path in protected}

        remote = self.workspace / "remotes" / "product-self.git"
        self.git(self.workspace, "init", "--bare", "--initial-branch=main", str(remote))
        self.git(target, "init", "--initial-branch=main")
        self.git_identity(target)
        self.git(target, "remote", "add", "origin", str(remote))
        self.git(target, "add", ".")
        self.git(target, "commit", "-m", "chore: adopt service continuity harness")
        self.git(target, "push", "-u", "origin", "main")

        initial_result, initial_status = self.cli_raw("status", "--root", str(target))
        initial_identified = (
            initial_status is not None
            and initial_status.get("project_id") == "project.service-continuity-product"
            and initial_result.returncode in {0, 3}
        )
        definition = self.work_definition(
            "work.release-guidance",
            "Complete release guidance",
            "Public installation and release guidance matches verified product behavior.",
            ["workspace.root"],
            ["repository.root"],
            ["README.md", "README.ko.md", "docs/**"],
            evidence=["test"],
        )
        definition["acceptance"] = [{
            "id": "scenario.release-guidance",
            "given": "The implementation and dogfood evidence are current",
            "when": "A maintainer follows the public guidance",
            "then": "The product can be installed, continued, and verified without hidden project knowledge",
            "failure": "Unsupported hosted-provider or publication claims remain explicit blockers",
        }]
        definition["first_failing_test"] = "test.release-guidance"
        input_path = self.workspace / "inputs" / "self-release-guidance.json"
        self.write(input_path, json.dumps(definition, sort_keys=True, indent=2) + "\n")
        defined = self.cli("work", "define", "--root", str(target), "--input", str(input_path), "--apply")
        self.expect_status(defined, "passed", "self release work definition")
        self.git(target, "add", ".harness/work/definitions")
        self.git(target, "commit", "-m", "docs: define release guidance work")
        self.git(target, "push", "origin", "main")
        started = self.cli(
            "work", "start", "--root", str(target), "--work-id", "work.release-guidance",
            "--claim-id", "claim.release-guidance", "--owner", "release-maintainer",
            "--branch", "docs/release-guidance", "--apply",
        )
        self.expect_status(started, "passed", "self release work reservation")
        status = self.cli("status", "--root", str(target))
        live = next((item for item in status.get("active_work", []) if item.get("id") == "work.release-guidance"), {})
        self.require(
            "self.product-adopt-and-live-work",
            before == after
            and initial_identified
            and (target / ".agents" / "skills" / "use-project-harness" / "SKILL.md").is_file()
            and status.get("provider", {}).get("confidence") == "confirmed"
            and live.get("owner") == "release-maintainer"
            and live.get("branch") == "docs/release-guidance",
            "the current product was non-destructively adopted, inspected, and coordinated through a real Git-local remote",
        )

    def seed_child(self, name: str, url: str) -> None:
        seed = self.workspace / "seeds" / name
        seed.mkdir(parents=True)
        self.git(seed, "init", "--initial-branch=main")
        self.git_identity(seed)
        self.configure_transport(seed, url)
        self.git(seed, "remote", "add", "origin", url)
        self.write(seed / "go.mod", f"module example.test/dogfood/{name}\n\ngo 1.22\n")
        if name == "backend":
            self.write(
                seed / "recovery.go",
                "package recovery\n\nfunc Decision(valid bool) string {\n\treturn \"denied\"\n}\n",
            )
        else:
            self.write(
                seed / "state.go",
                "package ui\n\nfunc RecoveryState(code string) string {\n\treturn \"ready\"\n}\n",
            )
        self.git(seed, "add", ".")
        self.git(seed, "commit", "-m", f"chore: initialize {name}")
        self.git(seed, "push", "-u", "origin", "main")

    def initialize_root(self) -> pathlib.Path:
        root = self.workspace / "orchestration"
        initialized = self.cli(
            "project", "init", "--root", str(root), "--id", "project.dogfood-service",
            "--name", "Dogfood Service", "--locale", "en", "--apply",
        )
        self.expect_status(initialized, "passed", "project initialization")
        self.git(root, "init", "--initial-branch=main")
        self.git_identity(root)
        for url in self.remotes:
            self.configure_transport(root, url)
        self.git(root, "config", "protocol.file.allow", "always")
        self.git(root, "remote", "add", "origin", ROOT_URL)
        self.git(root, "-c", "protocol.file.allow=always", "submodule", "add", FRONTEND_URL, "frontend")
        self.git(root, "-c", "protocol.file.allow=always", "submodule", "add", BACKEND_URL, "backend")
        for child, url in (("frontend", FRONTEND_URL), ("backend", BACKEND_URL)):
            self.git_identity(root / child)
            self.configure_transport(root / child, url)

        self.write(root / ".harness" / "workspaces.yaml", self.workspace_manifest())
        self.write(
            root / ".harness" / "work" / "provider.yaml",
            "schema_version: 1\nprovider: git-local\nlive_status_source: git-local\n"
            + "remote: " + json.dumps(str(self.remotes[ROOT_URL].resolve())) + "\n"
            + "coordination_branch: coordination\n",
        )
        self.write(root / ".harness" / "commands.yaml", self.root_commands())
        self.write(root / "contracts" / "business" / "account-recovery.md", self.contract_source())
        contract_bytes = (root / "contracts" / "business" / "account-recovery.md").read_bytes()
        self.write(root / "contracts" / "registry.yaml", self.contract_registry(contract_bytes))
        self.write(root / "specs" / "account-recovery.md", self.product_spec())
        self.write(root / "docs" / "release.txt", "account recovery service candidate\n")

        definitions = [
            self.work_definition(
                PARENT_WORK,
                "Coordinate account recovery",
                "Account recovery follows one approved business and failure contract.",
                ["workspace.root"],
                ["repository.root"],
                ["contracts/business/account-recovery.md"],
                contract_ids=[CONTRACT_ID],
                root_pointers=["workspace.frontend", "workspace.backend"],
                evidence=["test", "integration", "root-pointer"],
            ),
            self.work_definition(
                BACKEND_WORK,
                "Provide account recovery decisions",
                "The backend issues recovery only for valid proof.",
                ["workspace.backend"],
                ["repository.backend"],
                ["recovery/**"],
                parent=PARENT_WORK,
                dependencies=[PARENT_WORK],
                evidence=["test", "child-merge"],
            ),
            self.work_definition(
                FRONTEND_WORK,
                "Connect account recovery states",
                "The UI presents retry-safe recovery failure states.",
                ["workspace.frontend"],
                ["repository.frontend"],
                ["ui/**"],
                parent=PARENT_WORK,
                dependencies=[PARENT_WORK, BACKEND_WORK],
                evidence=["test", "child-merge"],
            ),
        ]
        inputs = self.workspace / "inputs"
        inputs.mkdir()
        for definition in definitions:
            path = inputs / f"{definition['id']}.json"
            self.write(path, json.dumps(definition, sort_keys=True, indent=2) + "\n")
            result = self.cli("work", "define", "--root", str(root), "--input", str(path), "--apply")
            self.expect_status(result, "passed", f"define {definition['id']}")

        audit = self.cli("context", "audit", "--root", str(root))
        self.expect_status(audit, "passed", "initial context audit")
        impact = self.cli("contract", "impact", "--root", str(root), "--id", CONTRACT_ID)
        self.expect_status(impact, "passed", "business contract impact")
        self.require(
            "contract.business-and-failure-approved",
            not audit.get("blockers")
            and any(
                item.get("code") == "contract.source" and CONTRACT_ID in item.get("refs", [])
                for item in impact.get("facts", [])
            ),
            "approved business and failure obligations are indexed with impact",
        )

        self.git(root, "add", ".")
        self.git(root, "commit", "-m", "chore: initialize service orchestration")
        self.git(root, "push", "-u", "origin", "main")
        inspect = self.cli("git", "inspect", "--root", str(root))
        facts = {(item.get("code"), item.get("message")) for item in inspect.get("facts", [])}
        self.require(
            "topology.root-and-two-submodules",
            ("git.submodules", "2") in facts,
            "root Git inspection found two initialized submodules",
        )
        return root

    def prove_claim_race_and_conflict(self, root: pathlib.Path) -> None:
        clones = []
        for owner in ("alex", "sam"):
            clone = self.workspace / "owners" / owner
            clone.parent.mkdir(exist_ok=True)
            self.clone(ROOT_URL, clone)
            self.git_identity(clone)
            self.persist_rewrites(clone)
            clones.append((owner, clone))

        def claim(owner_and_clone: tuple[str, pathlib.Path]) -> tuple[str, CommandResult, dict[str, Any] | None]:
            owner, clone = owner_and_clone
            result, value = self.cli_raw(
                "work", "start", "--root", str(clone), "--work-id", PARENT_WORK,
                "--claim-id", f"claim.account-recovery-{owner}", "--owner", owner,
                "--branch", "feature/account-recovery-contract", "--apply",
            )
            return owner, result, value

        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
            results = list(executor.map(claim, clones))
        winners = [owner for owner, result, value in results if result.returncode == 0 and value and value.get("status") == "passed"]
        self.require(
            "claim.race-single-owner",
            len(winners) == 1,
            "two simultaneous owners produced exactly one observable winner",
        )

        candidate = self.workspace / "inputs" / "semantic-conflict.yaml"
        self.write(
            candidate,
            textwrap.dedent(
                f"""\
                repository: repository.root
                paths: [docs/decisions/**]
                policy_ids: []
                scenario_ids: []
                contract_ids: [{CONTRACT_ID}]
                db_entities: []
                migration_slots: []
                ui_flows: []
                dependency_majors: []
                stable_ids: []
                root_pointer: false
                """
            ),
        )
        result, value = self.cli_raw("work", "conflict", "--root", str(root), "--candidate", str(candidate))
        codes = {item.get("code") for item in (value or {}).get("blockers", [])}
        self.require(
            "conflict.semantic-path-disjoint",
            result.returncode != 0 and "conflict.contract" in codes and "conflict.path-overlap" not in codes,
            "different files were blocked because the same business contract was being changed",
        )

    def start_child_work(self, root: pathlib.Path) -> None:
        for work_id, claim_id, owner, branch in (
            (BACKEND_WORK, "claim.account-recovery-backend", "alex", "feature/account-recovery-backend"),
            (FRONTEND_WORK, "claim.account-recovery-frontend", "sam", "feature/account-recovery-frontend"),
        ):
            result = self.cli(
                "work", "start", "--root", str(root), "--work-id", work_id,
                "--claim-id", claim_id, "--owner", owner, "--branch", branch, "--apply",
            )
            self.expect_status(result, "passed", f"start {work_id}")

        wrong_result, wrong_workspace = self.cli_raw(
            "work", "evidence", "--root", str(root), "--work-id", BACKEND_WORK,
            "--workspace", "workspace.frontend", "--command-id", "command.frontend-test", "--apply",
        )
        wrong_codes = {item.get("code") for item in (wrong_workspace or {}).get("blockers", [])}
        self.require(
            "evidence.wrong-workspace-blocked",
            wrong_result.returncode != 0 and "evidence.workspace-undeclared" in wrong_codes,
            "evidence could not be recorded from a workspace outside the work definition",
        )
        self.git(root, "add", ".harness/work")
        self.git(root, "commit", "-m", "chore: record account recovery ownership")
        self.git(root, "push", "origin", "main")

    def implement_backend(self, root: pathlib.Path) -> bool:
        child = root / "backend"
        branch = "feature/account-recovery-backend"
        self.git(child, "switch", "-c", branch)
        self.write(
            child / "recovery_test.go",
            textwrap.dedent(
                """\
                package recovery

                import "testing"

                func TestDecision(t *testing.T) {
                    if got := Decision(true); got != "issued" { t.Fatalf("valid proof: %s", got) }
                    if got := Decision(false); got != "denied" { t.Fatalf("invalid proof: %s", got) }
                }
                """
            ),
        )
        failing = self.command(["go", "test", "./..."], child, check=False)
        if failing.returncode != 0:
            self.tdd_failures += 1
        self.write(
            child / "recovery.go",
            textwrap.dedent(
                """\
                package recovery

                func Decision(valid bool) string {
                    if valid { return "issued" }
                    return "denied"
                }
                """
            ),
        )
        self.write(child / "service.txt", "account recovery provider\n")
        self.write(child / ".harness" / "commands.yaml", self.child_commands("backend"))
        passing = self.command(["go", "test", "./..."], child, check=False)
        self.git(child, "add", ".")
        self.git(child, "commit", "-m", "feat(recovery): issue verified recovery decisions")
        self.git(child, "push", "-u", "origin", branch)
        test = self.cli(
            "work", "evidence", "--root", str(child), "--work-id", BACKEND_WORK,
            "--workspace", "workspace.backend", "--command-id", "command.backend-test",
            "--artifact", "service=service.txt", "--apply",
        )
        merge = self.cli(
            "work", "evidence", "--root", str(child), "--work-id", BACKEND_WORK,
            "--workspace", "workspace.backend", "--command-id", "command.backend-merge", "--apply",
        )
        review = self.cli("work", "transition", "--root", str(child), "--work-id", BACKEND_WORK, "--target", "review", "--apply")
        self.git(child, "switch", "main")
        self.git(child, "merge", "--ff-only", branch)
        self.git(child, "push", "origin", "main")
        return failing.returncode != 0 and passing.returncode == 0 and all(value.get("status") == "passed" for value in (test, merge, review))

    def implement_frontend(self, root: pathlib.Path) -> bool:
        child = root / "frontend"
        branch = "feature/account-recovery-frontend"
        self.git(child, "switch", "-c", branch)
        self.write(
            child / "state_test.go",
            textwrap.dedent(
                """\
                package ui

                import "testing"

                func TestRecoveryState(t *testing.T) {
                    if got := RecoveryState("RATE_LIMITED"); got != "retry" { t.Fatalf("rate limited: %s", got) }
                    if got := RecoveryState(""); got != "ready" { t.Fatalf("ready: %s", got) }
                }
                """
            ),
        )
        failing = self.command(["go", "test", "./..."], child, check=False)
        if failing.returncode != 0:
            self.tdd_failures += 1
        self.write(
            child / "state.go",
            textwrap.dedent(
                """\
                package ui

                func RecoveryState(code string) string {
                    if code == "RATE_LIMITED" { return "retry" }
                    return "ready"
                }
                """
            ),
        )
        self.write(child / "ui.txt", "account recovery retry state\n")
        self.write(child / ".harness" / "commands.yaml", self.child_commands("frontend"))
        passing = self.command(["go", "test", "./..."], child, check=False)
        self.git(child, "add", ".")
        self.git(child, "commit", "-m", "feat(recovery): show retry-safe recovery state")
        self.git(child, "push", "-u", "origin", branch)
        test = self.cli(
            "work", "evidence", "--root", str(child), "--work-id", FRONTEND_WORK,
            "--workspace", "workspace.frontend", "--command-id", "command.frontend-test",
            "--artifact", "ui=ui.txt", "--apply",
        )
        merge = self.cli(
            "work", "evidence", "--root", str(child), "--work-id", FRONTEND_WORK,
            "--workspace", "workspace.frontend", "--command-id", "command.frontend-merge", "--apply",
        )
        review = self.cli("work", "transition", "--root", str(child), "--work-id", FRONTEND_WORK, "--target", "review", "--apply")
        self.git(child, "switch", "main")
        self.git(child, "merge", "--ff-only", branch)
        self.git(child, "push", "origin", "main")
        return failing.returncode != 0 and passing.returncode == 0 and all(value.get("status") == "passed" for value in (test, merge, review))

    def integrate_child_pointers(self, root: pathlib.Path) -> None:
        self.git(root, "add", "frontend", "backend")
        self.git(root, "commit", "-m", "build: integrate account recovery workspaces")
        self.git(root, "push", "origin", "main")
        self.remove_transport_rewrites(root)
        self.remove_transport_rewrites(root / "frontend")
        self.remove_transport_rewrites(root / "backend")
        inspect = self.cli("git", "inspect", "--root", str(root))
        pointer_values = [item.get("message") for item in inspect.get("facts", []) if item.get("code") == "git.submodule.pointer-mismatch"]
        self.require(
            "submodule.pointers-exact",
            pointer_values == ["false", "false"],
            "both child HEADs equal the root-recorded pointers",
        )

    def prove_parent_worktree_and_evidence(self, root: pathlib.Path) -> None:
        target = self.workspace / "parent-worktree"
        created = self.cli(
            "git", "worktree", "--root", str(root), "--branch", "feature/account-recovery-contract",
            "--base", "main", "--target", str(target), "--apply",
        )
        self.require(
            "worktree.conventional-isolation",
            created.get("status") == "passed" and self.git_output(target, "branch", "--show-current") == "feature/account-recovery-contract",
            "a conventional branch was created in a verified isolated worktree",
        )
        for command_id, artifact in (
            ("command.root-test", ["--artifact", "manifest=docs/release.txt"]),
            ("command.root-integration", []),
            ("command.root-pointer", []),
        ):
            result = self.cli(
                "work", "evidence", "--root", str(target), "--work-id", PARENT_WORK,
                "--workspace", "workspace.root", "--command-id", command_id, *artifact, "--apply",
            )
            self.expect_status(result, "passed", command_id)
        review = self.cli("work", "transition", "--root", str(target), "--work-id", PARENT_WORK, "--target", "review", "--apply")
        self.expect_status(review, "passed", "parent review")
        source = target / ".harness" / "local" / "evidence" / PARENT_WORK
        destination = root / ".harness" / "local" / "evidence" / PARENT_WORK
        shutil.copytree(source, destination, dirs_exist_ok=True)
        self.git(root, "worktree", "remove", str(target))

    def prove_integration(self, root: pathlib.Path) -> None:
        planned = self.cli("integrate", "plan", "--root", str(root), "--apply")
        self.expect_status(planned, "passed", "integration plan")
        ordered_work = [item.get("refs", [""])[0] for item in planned.get("facts", []) if item.get("code") == "integrate.step"]
        backend_index = ordered_work.index(BACKEND_WORK) if BACKEND_WORK in ordered_work else -1
        frontend_index = ordered_work.index(FRONTEND_WORK) if FRONTEND_WORK in ordered_work else -1
        self.require(
            "integration.provider-before-consumer",
            0 <= backend_index < frontend_index,
            "dependency order placed the backend provider before the frontend consumer",
        )
        verified = self.cli("integrate", "verify", "--root", str(root))
        self.expect_status(verified, "passed", "integration verification")
        for work_id in (BACKEND_WORK, FRONTEND_WORK):
            integrated = self.cli("work", "transition", "--root", str(root), "--work-id", work_id, "--target", "integrated", "--apply")
            self.expect_status(integrated, "passed", f"integrate {work_id}")
            done = self.cli("work", "finish", "--root", str(root), "--work-id", work_id, "--apply")
            self.expect_status(done, "passed", f"finish {work_id}")
        parent_integrated = self.cli("work", "transition", "--root", str(root), "--work-id", PARENT_WORK, "--target", "integrated", "--apply")
        self.expect_status(parent_integrated, "passed", "integrate parent")
        parent_done = self.cli("work", "finish", "--root", str(root), "--work-id", PARENT_WORK, "--apply")
        self.expect_status(parent_done, "passed", "finish parent")

    def prove_release(self, root: pathlib.Path) -> None:
        prepared = self.cli(
            "release", "prepare", "--root", str(root), "--release-version", "1.0.0",
            "--work", PARENT_WORK, "--work", BACKEND_WORK, "--work", FRONTEND_WORK, "--apply",
        )
        self.expect_status(prepared, "passed", "release prepare")
        validation = self.workspace / "user-validation.txt"
        self.write(validation, "same candidate passed user journey verification\n")
        validated = self.cli("release", "validate", "--root", str(root), "--evidence", str(validation), "--confirm", "--apply")
        self.expect_status(validated, "passed", "release validation")
        verified = self.cli("release", "verify", "--root", str(root))
        self.require(
            "release.exact-candidate",
            verified.get("status") == "passed" and any(item.get("code") == "release.candidate-digest" for item in verified.get("evidence", [])),
            "technical and user evidence verified against the same candidate digest",
        )

        candidate = root / ".harness" / "local" / "release" / "candidate.json"
        original = candidate.read_bytes()
        value = json.loads(original)
        digest = value["digest"]
        value["digest"] = digest[:-1] + ("0" if digest[-1] != "0" else "1")
        self.write(candidate, json.dumps(value, sort_keys=True, indent=2) + "\n")
        result, tampered = self.cli_raw("release", "verify", "--root", str(root))
        candidate.write_bytes(original)
        blocker_codes = {item.get("code") for item in (tampered or {}).get("blockers", [])}
        self.require(
            "release.tamper-blocked",
            result.returncode != 0 and "release.candidate-changed" in blocker_codes,
            "a changed candidate digest was rejected",
        )

    def prove_clone_recovery(self, root: pathlib.Path) -> None:
        clone = self.workspace / "clean-clone"
        self.clone(ROOT_URL, clone, recurse=True)
        self.git_identity(clone)
        self.persist_rewrites(clone)
        for child, url in ((clone / "frontend", FRONTEND_URL), (clone / "backend", BACKEND_URL)):
            self.git_identity(child)
            self.configure_transport(child, url)
        status = self.cli("status", "--root", str(clone / "frontend"))
        issue_codes = {item.get("code") for item in status.get("issues", [])}
        owners = {item.get("id"): item.get("owner") for item in status.get("active_work", [])}
        self.require(
            "clone.context-recovered",
            status.get("project_id") == "project.dogfood-service"
            and status.get("current_workspace_id") == "workspace.frontend"
            and "workspace.pointer-mismatch" not in issue_codes
            and status.get("provider", {}).get("confidence") == "confirmed"
            and not owners,
            "a recursive clone recovered the root, exact pointers, and completed live work",
        )
        next_actions = status.get("next_actions", [])
        self.require(
            "clone.next-action-recovered",
            len(next_actions) == 1 and next_actions[0].get("code") == "work.define-next",
            "combined status produced one safe next action without replaying the conversation",
        )

        backend = clone / "backend"
        self.git(backend, "switch", "-c", "feature/local-only-check")
        self.write(backend / "local-only.txt", "unpublished local context\n")
        self.git(backend, "add", "local-only.txt")
        self.git(backend, "commit", "-m", "test: record local-only state")
        local_result, local_status = self.cli_raw("status", "--root", str(backend))
        local_codes = {item.get("code") for item in (local_status or {}).get("issues", [])}
        self.require(
            "clone.local-only-unrecoverable",
            local_result.returncode != 0 and "workspace.local-only" in local_codes,
            "an unpublished child commit was classified as local-only",
        )
        self.require(
            "clone.pointer-drift-detected",
            "workspace.pointer-mismatch" in local_codes,
            "a child HEAD that differed from the root gitlink was detected before integration",
        )

    def workspace_manifest(self) -> str:
        return textwrap.dedent(
            f"""\
            schema_version: 1
            project_id: project.dogfood-service
            root_remote: {ROOT_URL}
            workspaces:
              - id: workspace.root
                kind: root
                path: .
                repository: repository.root
                remote: {ROOT_URL}
                responsibilities: [orchestration, contracts, integration]
                dependencies: []
                commands_path: .harness/commands.yaml
              - id: workspace.backend
                kind: submodule
                path: backend
                repository: repository.backend
                remote: {BACKEND_URL}
                responsibilities: [account-recovery-provider]
                dependencies: [workspace.root]
                commands_path: .harness/commands.yaml
              - id: workspace.frontend
                kind: submodule
                path: frontend
                repository: repository.frontend
                remote: {FRONTEND_URL}
                responsibilities: [account-recovery-ui]
                dependencies: [workspace.root, workspace.backend]
                commands_path: .harness/commands.yaml
            """
        )

    def root_commands(self) -> str:
        return textwrap.dedent(
            """\
            schema_version: 1
            workspace_id: workspace.root
            commands:
              - id: command.root-test
                kind: test
                argv: [git, diff, --check, HEAD]
                timeout_seconds: 30
              - id: command.root-integration
                kind: integration
                argv: [git, diff, --check, HEAD]
                timeout_seconds: 30
              - id: command.root-pointer
                kind: root-pointer
                argv: [git, diff, --check, HEAD]
                timeout_seconds: 30
            """
        )

    def child_commands(self, name: str) -> str:
        return textwrap.dedent(
            f"""\
            schema_version: 1
            workspace_id: workspace.{name}
            commands:
              - id: command.{name}-test
                kind: test
                argv: [go, test, ./...]
                timeout_seconds: 120
              - id: command.{name}-merge
                kind: child-merge
                argv: [go, test, ./...]
                timeout_seconds: 120
            """
        )

    def contract_source(self) -> str:
        return textwrap.dedent(
            f"""\
            ---
            schema_version: 1
            id: {CONTRACT_ID}
            kind: business
            status: approved
            revision: 1
            refs: []
            ---

            # Account recovery obligations

            Purpose: restore access only after valid proof.

            Business rule: valid proof issues one recovery decision; invalid proof never restores access.

            Failure behavior: RATE_LIMITED is retryable and the UI must not report success.

            Non-goal: this flow never changes account ownership.
            """
        )

    def contract_registry(self, source: bytes) -> str:
        digest = hashlib.sha256(source).hexdigest()
        return textwrap.dedent(
            f"""\
            schema_version: 1
            contracts:
              - id: {CONTRACT_ID}
                kind: business
                status: approved
                revision: 1
                source: business/account-recovery.md
                compatibility: additive
                providers: [workspace.backend]
                consumers: [workspace.frontend]
                product_ids: [product.account-recovery]
                scenario_ids: []
                data_ids: []
                ui_ids: []
                migration_ids: []
                work_ids: [{PARENT_WORK}, {BACKEND_WORK}, {FRONTEND_WORK}]
                test_ids: []
                refs: []
                fingerprint: sha256:{digest}
            """
        )

    def product_spec(self) -> str:
        return textwrap.dedent(
            f"""\
            ---
            schema_version: 1
            id: product.account-recovery
            kind: product
            status: approved
            revision: 1
            refs: [{CONTRACT_ID}]
            ---

            Members recover account access through verified proof and explicit retry-safe failures.
            """
        )

    def work_definition(
        self,
        work_id: str,
        title: str,
        outcome: str,
        workspaces: list[str],
        repositories: list[str],
        paths: list[str],
        *,
        parent: str | None = None,
        dependencies: list[str] | None = None,
        contract_ids: list[str] | None = None,
        root_pointers: list[str] | None = None,
        evidence: list[str] | None = None,
    ) -> dict[str, Any]:
        value: dict[str, Any] = {
            "schema_version": 1,
            "id": work_id,
            "readiness": "ready",
            "title": title,
            "outcome": outcome,
            "acceptance": [{
                "id": "scenario." + work_id.removeprefix("work."),
                "given": "The approved account recovery contract exists",
                "when": "The owned change is integrated",
                "then": outcome,
                "failure": "Unsafe or stale behavior is rejected",
            }],
            "refs": [],
            "workspaces": workspaces,
            "scope": {
                "repositories": repositories,
                "paths": paths,
                "policy_ids": [],
                "scenario_ids": [],
                "contract_ids": contract_ids or [],
                "db_entities": [],
                "migration_slots": [],
                "ui_flows": [],
                "dependency_majors": [],
                "root_pointers": root_pointers or [],
            },
            "dependencies": dependencies or [],
            "merge_order": workspaces,
            "first_failing_test": "test." + work_id.removeprefix("work."),
            "evidence": {
                "kinds": evidence or ["test"],
                "integration_required": bool(set(evidence or []) & {"integration", "child-merge", "root-pointer"}),
                "user_validation": False,
                "migration_required": False,
                "rollback_required": False,
            },
        }
        if parent:
            value["parent_id"] = parent
        return value

    def cli(self, *args: str) -> dict[str, Any]:
        result, value = self.cli_raw(*args)
        if value is None:
            raise ScenarioError(f"CLI returned no JSON for {' '.join(args[:2])}: {result.stderr.strip()}")
        if result.returncode != 0:
            raise ScenarioError(f"CLI blocked {' '.join(args[:2])}: {self.codes(value)}")
        return value

    def cli_raw(self, *args: str) -> tuple[CommandResult, dict[str, Any] | None]:
        self.cli_calls += 1
        result = self.command([str(self.binary), *args, "--json"], self.workspace, check=False)
        value = None
        if result.stdout.strip():
            try:
                value = json.loads(result.stdout)
            except json.JSONDecodeError as error:
                raise ScenarioError(f"invalid CLI JSON for {' '.join(args[:2])}: {error}") from error
        return result, value

    def git(self, cwd: pathlib.Path, *args: str) -> CommandResult:
        self.git_calls += 1
        return self.command(["git", *args], cwd)

    def git_output(self, cwd: pathlib.Path, *args: str) -> str:
        return self.git(cwd, *args).stdout.strip()

    def command(self, argv: list[str], cwd: pathlib.Path, *, check: bool = True) -> CommandResult:
        environment = os.environ.copy()
        environment.update({"GIT_TERMINAL_PROMPT": "0", "GIT_ALLOW_PROTOCOL": "https:file"})
        environment["GIT_CONFIG_COUNT"] = str(len(self.remotes))
        for index, (url, remote) in enumerate(self.remotes.items()):
            environment[f"GIT_CONFIG_KEY_{index}"] = f"url.{remote.resolve().as_uri()}.insteadOf"
            environment[f"GIT_CONFIG_VALUE_{index}"] = url
        completed = subprocess.run(
            argv,
            cwd=cwd,
            env=environment,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=180,
            check=False,
        )
        result = CommandResult(completed.returncode, completed.stdout, completed.stderr)
        if check and completed.returncode != 0:
            label = pathlib.Path(argv[0]).name
            raise ScenarioError(f"{label} command failed ({completed.returncode}): {completed.stderr.strip() or completed.stdout.strip()}")
        return result

    def clone(self, url: str, target: pathlib.Path, *, recurse: bool = False) -> None:
        argv = ["git", "-c", "protocol.file.allow=always"]
        for source_url, remote in self.remotes.items():
            argv.extend(["-c", f"url.{remote.resolve().as_uri()}.insteadOf={source_url}"])
        argv.extend(["clone"])
        if recurse:
            argv.append("--recurse-submodules")
        argv.extend([url, str(target)])
        self.git_calls += 1
        self.command(argv, self.workspace)

    def persist_rewrites(self, repository: pathlib.Path) -> None:
        self.git(repository, "config", "protocol.file.allow", "always")
        for url in self.remotes:
            self.configure_transport(repository, url)

    def configure_transport(self, repository: pathlib.Path, url: str) -> None:
        remote = self.remotes[url].resolve().as_uri()
        self.git(repository, "config", f"url.{remote}.insteadOf", url)

    def remove_transport_rewrites(self, repository: pathlib.Path) -> None:
        for remote in self.remotes.values():
            self.git_calls += 1
            self.command(
                ["git", "config", "--remove-section", f"url.{remote.resolve().as_uri()}"],
                repository,
                check=False,
            )

    def git_identity(self, repository: pathlib.Path) -> None:
        self.git(repository, "config", "user.name", "Dogfood User")
        self.git(repository, "config", "user.email", "dogfood@example.test")

    def write(self, path: pathlib.Path, content: str) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def expect_status(self, value: dict[str, Any], status: str, label: str) -> None:
        if value.get("status") != status:
            raise ScenarioError(f"{label} returned {value.get('status')}: {self.codes(value)}")

    def require(self, code: str, condition: bool, evidence: str) -> None:
        self.assertions.append({"code": code, "status": "passed" if condition else "failed", "evidence": evidence})
        if not condition:
            raise ScenarioError(f"assertion failed: {code}")

    @staticmethod
    def codes(value: dict[str, Any]) -> str:
        return ",".join(item.get("code", "") for key in ("blockers", "warnings") for item in value.get(key, []))

    def report(self, error: Exception | None) -> dict[str, Any]:
        return {
            "schema_version": 1,
            "scenario": SCENARIO,
            "status": "failed" if error else "passed",
            "assertions": self.assertions,
            "measurements": {
                "required_assertions": len(self.assertions),
                "observed_failing_tests": self.tdd_failures,
                "cli_calls": self.cli_calls,
                "git_calls": self.git_calls,
            },
            "error": str(error) if error else "",
        }


def parser() -> argparse.ArgumentParser:
    value = argparse.ArgumentParser(description=__doc__)
    value.add_argument("--binary", required=True)
    value.add_argument("--workspace", required=True)
    value.add_argument("--output", required=True)
    return value


def main() -> int:
    args = parser().parse_args()
    dogfood = Dogfood(pathlib.Path(args.binary), pathlib.Path(args.workspace), pathlib.Path(args.output))
    error: Exception | None = None
    try:
        dogfood.run()
    except Exception as caught:  # report the first dependent scenario failure deterministically
        error = caught
    report = dogfood.report(error)
    dogfood.output.parent.mkdir(parents=True, exist_ok=True)
    dogfood.output.write_text(json.dumps(report, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, sort_keys=True))
    if error:
        print(f"dogfood failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
