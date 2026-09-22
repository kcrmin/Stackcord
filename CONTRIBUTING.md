# Contributing

## Development contract

- Protect `main`; use short-lived `<type>/<description>` branches and Conventional Commits without AI markers.
- Run context and conflict preflight before implementation that changes shared meaning or boundaries.
- Use test-first development for behavior, bug, contract, migration, security, UI interaction, Git mutation, and lifecycle changes.
- Keep generated JSON derived from canonical source; do not edit checkpoints directly.
- Preserve framework, language, database, cloud, and Git-host neutrality.
- Do not add telemetry, an automatically installed service, a required hosted account, hidden Git mutation, or a mandatory external adapter. The opt-in foreground runtime is supported.

## Local verification

For workflow triggers, required checks, repository setup and draft releases, see
the [CI/CD guide](docs/guides/ci-cd-en.md) ([한국어](docs/guides/ci-cd-ko.md)).

During development, run the changed package or validator first:

```sh
cd cli
go test ./internal/harness/<package> -run <test> -v
```

Before a pull request, run the deterministic repository checks:

```sh
cd core
go test -race ./...
go vet ./...
cd ../cli
go test ./...
go vet ./...
cd ..
python3 scripts/validate_docs.py
sh scripts/validate-plugin.sh
python3 scripts/validate_release_config.py
python3 scripts/security_scan.py .
```

Run `go test -race ./...` for concurrency changes, scheduled verification, and
release checks rather than for every local edit.

CI and normal releases do not invoke Codex. If Skill behavior changes, explicitly
run one relevant scenario:

```sh
python3 scripts/run_agent_eval.py \
  --scenarios evals/agent-behavior/scenarios.yaml \
  --rubric evals/agent-behavior/rubric.yaml \
  --output .harness/local/evals/skill-change \
  --scenario continue-after-clean-clone
```

The complete nine-scenario evaluation is a manual audit and requires both `--all`
and `--allow-external-research`. It consumes AI tokens and is never an automatic
CI gate.

A pull request describes objective, stable spec/contract references, conflict
scope, implementation and merge order, tests/evidence, risk, rollback, and
generated changes. Breaking contracts require side-by-side versions or an
explicitly approved maintenance plan.

## Review

Review product semantics before code style. Verify actual Git/submodule state, scope ownership, compatibility, failure behavior, tests, security, portability, documentation parity, and rollback. Do not accept completion claims without fresh command output.

## Module boundary

`core/` must never import the CLI, console or harness. Its public API is the
coordination contract. `cli/internal/harness/` contains existing optional project
workflows. Runtime mutations do not confer Git or product approval. Run both Go
modules when changing shared behavior; use the root README for the new entrypoint.
