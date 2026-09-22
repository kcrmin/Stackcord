# Coordination core

Status: implementation authorized by the repository owner on 2026-09-22.
This decision supersedes the full-stack-first positioning in historical specifications.

## Product and boundaries

Stackcord is an Apache-2.0 coordination library and runtime for AI workers. Its
primary job is to preserve tasks across worker/session replacement, exchange
bounded structured context without Git messages, and show the same facts to people.
Git continues to own source, product decisions, contracts and harness configuration.
The runtime owns live execution state; it never treats completion reports as code
verification or policy approval. Existing harness checks remain available.

One repository is appropriate: there is one maintainer/release boundary and no
organization ownership problem to solve. Core must not import CLI, GUI or harness.
Use a separate Go module under core/, with coordination and httpapi packages.
Keep the existing implementation under cli/internal/harness/, reachable through
`stackcord harness`. Preserve existing commands as hidden compatibility aliases.
A small optional web console consumes the runtime API without a model call.

## Retain, separate, remove, defer

| Capability | Decision | Reason |
| --- | --- | --- |
| Task identity, dependencies, checkpoints, ownership | Core | Work outlives sessions |
| HTTP requests and cursor-based event reads | Core transport | No Git message commit/polling |
| Required context, artifact references, byte budget | Core | Deliver only task-relevant data; never silently truncate obligations |
| Durable state and consistent backup | Core | Live state is authoritative, not a disposable cache |
| CLI | Adapter | Automation and existing AI tools |
| GUI | Optional adapter in the same repository | Humans inspect and control the same state |
| Git context/conflicts/contracts/release evidence | Harness extension | Valuable for code projects, not a universal agent requirement |
| QDD, DBML, external UI baseline, product governance | Harness extension | Domain/workflow-specific policy |
| Signed Git mailbox and existing host runners | Compatibility extension | Preserve installed workflows and receipts |
| Full-stack/QDD-first onboarding and keyword-only README tests | Remove from default | Conceals the focused core and locks prose to obsolete positioning |
| Automatic execution, quota reset, inferred token balance | Not claimed | No portable provider contract; report known usage only |
| NATS, A2A, distributed broker replication, desktop shell | Defer | No current need to pay operational/maintenance cost |

## Durable protocol

A task belongs to a project, not to a worker. Mutations have a caller-supplied
idempotency key. The same key with different contents is rejected. Task state,
compact event and idempotency receipt commit in one bbolt transaction. Writes are
synced by the database. One process owns a database file; workers connect by HTTP.
No event retention deletion is performed. Backups are consistent database copies;
restore requires stopping the runtime. The database should live outside Git.

Create -> ready; claim -> running; checkpoint renews lease and saves handoff;
pause -> paused; complete -> completed; fail -> failed. Claim can acquire a ready,
paused or expired running task after its dependencies complete. Approval-required
pauses are not automatically claimable. Explicit requeue changes paused/failed to
ready. Every running update must match owner, session and monotonically increasing
epoch and have an unexpired lease. Old sessions cannot update after reassignment.
Lease expiry detects eligibility, not whether a process stopped. External effects
still need isolation/idempotency and Git verification before integration.

Checkpoint fields: completed summary, next step, blockers, artifact references.
Task definition fields: goal, constraints, input references, dependency task IDs,
project-scoped addressing. Required context is never silently shortened. If its
serialized form exceeds the requested byte budget, return an explicit error. Byte
budgets are not tokenizer-exact limits. Events contain IDs and state, not duplicate
context bodies. Resume returns current task/dep facts, not an entire chat history.
Usage is optional worker-reported input/output token increments, counted once by
mutation ID. Unknown usage remains unknown; no savings percentage is claimed.

## HTTP and interfaces

The library handler accepts an explicit bearer token and optional exact Host;
CLI binds loopback only. Remote use is via an authenticated SSH tunnel or an
operator-provided HTTPS reverse proxy; no unauthenticated LAN listener. No shell
commands are accepted by the runtime. API requests are bounded strict JSON.
Endpoints: POST /v1/apply, GET /v1/tasks, GET /v1/context, GET /v1/events,
GET /v1/projects, GET /v1/backup. Project is mandatory for task reads. All routes
require authentication. No cross-origin browser access. The console is optional
and renders untrusted content with textContent. Bundled console access is loopback/SSH-only; TLS-terminating reverse proxies
are for API clients, not an implicitly supported browser deployment. CLI credentials come from a file
or environment, never persistent browser storage or command-line token flags.

## Acceptance

- A pauses for quota; B claims and completes; A's stale update is rejected.
- Close/reopen preserves work, checkpoint, event cursor and idempotency receipts.
- Lost-response retries do not duplicate usage, tasks or events.
- Expired workers cannot overwrite a new owner, even with the same worker name.
- Dependency failures block claims; an explicit retry preserves the checkpoint.
- Project-scoped events paginate without loss and contain no other project data.
- A small context fails explicitly when mandatory information will not fit.
- HTTP has the same semantics as the embedded library, with no Git installed.
- Unauthorized/oversized/unknown-field/cross-origin requests do not mutate state.
- Existing harness tests and source builds continue passing after relocation.

## Limits

No automatic LLM execution or provider quota management in the new runtime. Any
AI tool can use its CLI/HTTP interface; existing automatic runners stay in the Git
harness. No implicit migration of legacy Git mailbox history. Retain that history
and begin new tasks in the new runtime. No automatic Git export; checkpoint
references can identify exact Git commits, whose verification belongs to harness.

## Verification record — 2026-09-22

Validated locally on macOS; these results do not constitute a production release.

- Core: `go test -race ./...` and `go vet ./...` passed, including ownership
  fencing, retry idempotency, reopen/backup recovery, dependency checks, bounded
  context and authenticated HTTP. Its dependency graph contains no CLI/harness.
- CLI: `go test ./...` and `go vet ./...` passed. Race checks passed for the new
  runtime/console and existing harness channel, dashboard and control center.
- The opt-in `TestProductionE2EMultiRepositoryContinuity` passed after relocation.
  This Python-based scenario requires the supported Python runtime; macOS's
  bundled Python 3.9 cannot run its existing `Path.write_text(newline=...)` calls.
- Three JavaScript regression tests exercise the shipped console's project-switch
  race, disconnect during an in-flight response and preservation of form drafts.
  Live browser checks covered login, task creation, EN/KO and readable handoff.
- Live HTTP checks covered Alice pausing for quota, Bob claiming the same task,
  and rejection of Alice's stale completion with HTTP 409.
- Repository validation passed: 76 Python tests, four baseline-evaluation tests,
  seven strict-release profile tests, 17 EN/KO documentation pairs, plugin/schema,
  release configuration, secret scanning and actionlint.
- CGO-free builds passed for Darwin amd64/arm64, Windows amd64/arm64 and Linux
  amd64. Windows/Linux execution was not tested locally.
- `TestResumeSize` retained a roughly 1,992-byte current context after 50
  checkpoints; the compact event feed was 7,118 bytes. The test checks that current
  context does not grow with history. These are fixture byte counts, not measured
  LLM token savings or a comparison against another product.

Independent review findings about payload headroom and project-switch races were
fixed with regression coverage. The reverse-proxy browser deployment limitation
is explicit. No provider/model benchmark, new module tag, release or merge is part
of this change.
