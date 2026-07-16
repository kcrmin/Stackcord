# Contributing

## Development contract

- Protect `main`; use short-lived `<type>/<description>` branches and Conventional Commits without AI markers.
- Run context and conflict preflight before implementation that changes shared meaning or boundaries.
- Use test-first development for behavior, bug, contract, migration, security, UI interaction, Git mutation, and lifecycle changes.
- Keep generated JSON derived from canonical source; do not edit checkpoints directly.
- Preserve framework, language, database, cloud, and Git-host neutrality.
- Do not add telemetry, a daemon, a required account, hidden Git mutation, or a mandatory external adapter.

## Local verification

```sh
cd cli
go test -race ./...
go vet ./...
cd ..
python3 scripts/validate_docs.py
sh scripts/validate-plugin.sh
python3 scripts/validate_release_config.py
python3 scripts/security_scan.py .
```

Use `go test ./internal/<package> -run <test> -v` to show the red and green cycle. A pull request describes objective, stable spec/contract references, conflict scope, implementation and merge order, tests/evidence, risk, rollback, and generated changes. Breaking contracts require side-by-side versions or an explicitly approved maintenance plan.

## Review

Review product semantics before code style. Verify actual Git/submodule state, scope ownership, compatibility, failure behavior, tests, security, portability, documentation parity, and rollback. Do not accept completion claims without fresh command output.
