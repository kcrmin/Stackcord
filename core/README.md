# Stackcord core

Apache-2.0 Go library for durable coordination. This module has no dependency on
Stackcord's CLI, GUI, Git harness or model providers. Its storage dependency is
[etcd bbolt](https://github.com/etcd-io/bbolt), a transactional pure-Go database.

This is an unreleased source module. In this monorepo the CLI uses a local replace
directive. A module release would require its own `core/v…` tag; none is created by
this change. Do not assume `v1.0.0` of the repository contains this module.

```go
store, err := coordination.Open("/private/local/path/state.db")
if err != nil { return err }
defer store.Close()

task, err := store.Apply(coordination.Mutation{
    ID: "request-001", Project: "service", Task: "api", Kind: "create",
    Definition: &coordination.Definition{
        Goal: "Implement the approved endpoint",
        Constraints: []string{"Preserve the existing response contract"},
        Inputs: []coordination.Reference{{URI: "git:commit:path", Digest: "sha256:…"}},
    },
})
```

Imports: `github.com/kcrmin/Stackcord/core/coordination` and, when needed,
`github.com/kcrmin/Stackcord/core/httpapi`.

## State protocol

`create` → ready; `claim` → running; `checkpoint` renews ownership;
`pause` → paused; `complete` → completed; `fail` → failed.

Claim specifies `owner`, a **new session ID**, and a 1–3600 second lease. Every
running update carries the returned `owner`, `session` and `epoch`. Expired leases
cannot write. The next owner gets a new epoch. Dependencies must have completed.
A quota pause is claimable; an `approval_required` pause needs explicit `requeue`
with `expected_revision`. Requeue is also available for failed tasks. It preserves
the checkpoint and does not certify approval, kill processes or undo side effects.

Every mutation has a project-scoped idempotency ID. Retry lost requests with the
**same payload and ID**. A successful retry returns the original mutation result,
which may now be historical: use `Context` before doing new work. A different
payload with the same ID is rejected. Failed mutations have no committed effect.

`Usage` is an optional input/output token increment, not a provider balance. Nil
means no reported usage. Partial observations do not imply full task cost.

## Queries and durability

- `Context(project, task, maxBytes)`: current task and direct dependency artifacts;
  returns an explicit error if mandatory context exceeds the byte budget.
- `Events(project, after, limit)`: ordered compact changes, up to 100 per page;
  `next` advances only through returned events. No implicit acknowledgement/deletion.
- `Tasks(project)`: human overview, bounded to 1000 tasks / 4 MiB. Use task context
  and event cursors when the overview exceeds that bound.
- `Projects()`: up to 1000 projects; intended for a trusted team's local runtime.
- `Backup(writer)`: consistent binary database snapshot. Restore while stopped.

One process owns the file. Transactions persist task state, event and retry receipt
together. The database is authoritative and must be backed up; Git clone cannot
recover it. State is unencrypted on disk; use private storage/OS access controls.
No event deletion, retention compaction or automatic Git export is implemented.
Limits bound requests and responses, not lifetime disk usage.

## HTTP

`httpapi.New(store, token, host)` is an authenticated `http.Handler`;
`httpapi.NewClient(endpoint, token)` uses HTTPS or loopback HTTP and refuses redirects.
The handler does not open a listener, terminate TLS, or import UI assets.

`POST /v1/apply` takes `coordination.Mutation`. GET routes are `/v1/projects`,
`/v1/tasks?project=…`, `/v1/context?project=…&task=…&max_bytes=16384`,
`/v1/events?project=…&after=0&limit=100`, and `/v1/backup`.
All require bearer auth. Mutation input is strict JSON, at most 64 KiB. Definitions
are limited to 32 KiB and checkpoints to 24 KiB, reserving ownership metadata
headroom. Aggregate context accepts a budget up to 1 MiB for dependency artifacts.
The bundled console is loopback/SSH-only. Browser deployments of the library must
serve HTTPS directly with a matching Host. TLS-terminating proxies need an explicit
origin policy outside this handler; forwarded headers are not implicitly trusted.

A token grants all projects and operator actions in this runtime. Worker labels
are coordination identities, not cryptographic authorization principals. Keep
untrusted tenants in separate runtimes. A lease fences state writes, not external
filesystem or service effects. Reported completion is not Git/test evidence.
