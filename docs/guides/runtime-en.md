# Runtime and worker continuity

## Start and connect

The runtime in current source is independent of Git and the optional project
harness. Build the checkout as shown in the root README; it is not in the older
`v1.0.0` release. Use a private directory outside your source repository.

On macOS/Linux, for example:

```sh
mkdir -p "$HOME/.config/stackcord"
chmod 700 "$HOME/.config/stackcord"
(umask 077; openssl rand -hex 32 > "$HOME/.config/stackcord/token")
stackcord serve --db "$HOME/.config/stackcord/state.db" --token-file "$HOME/.config/stackcord/token" --ui
```

Create the credential once; do not overwrite an existing team's credential.
On Windows, create a random credential in a file readable only by your user and
pass its path with the same `--token-file` flag. The server requires at least
32 characters; use a cryptographic random generator, not a memorable password.

The server listens only on `127.0.0.1:7331`; `--port 0` chooses an available port.
Open the printed address for the optional UI and enter the credential. The CLI
also accepts `STACKCORD_TOKEN`; a token file takes precedence. Credential values
are not accepted as command-line flags or printed in the server URL.

Each CLI request below uses `--token-file /private/path/token`. Alternatively set
`STACKCORD_TOKEN` securely in the calling process. `--endpoint` defaults to
`http://127.0.0.1:7331`. Runtime results are JSON regardless of the legacy `--json` flag.

## A pauses and B takes over

Save each JSON object to a file and pass it using `task apply`. The same API is
available to custom model adapters; no special model or prompt format is required.
Use a new mutation ID per action and reuse it only when retrying that exact action.

Create `create.json`:

```json
{"id":"create-api","project":"shop","task":"api","kind":"create","definition":{"goal":"Implement the approved checkout endpoint","constraints":["Preserve the payment contract"]}}
```

```sh
stackcord task apply --input create.json --token-file /private/path/token
```

Claim as A (`claim-a.json`):

```json
{"id":"claim-a","project":"shop","task":"api","kind":"claim","owner":"alice","session":"alice-session-1","lease_seconds":300}
```

The result includes `epoch: 1`. Store the returned owner/session/epoch. While
running, send `checkpoint` with that tuple and `lease_seconds` before expiry.
If the provider reports a quota limit, save a pause (`pause-a.json`):

```json
{"id":"pause-a","project":"shop","task":"api","kind":"pause","owner":"alice","session":"alice-session-1","epoch":1,"reason":"quota","checkpoint":{"summary":"Request schema implemented","next":"Add validation tests","artifacts":[{"uri":"git:COMMIT:path/to/schema","digest":"sha256:CONTENT_HASH"}]},"usage":{"input_tokens":1200,"output_tokens":300}}
```

Claim as B (`claim-b.json`):

```json
{"id":"claim-b","project":"shop","task":"api","kind":"claim","owner":"bob","session":"bob-session-1","lease_seconds":300}
```

Apply those files in order with the same command. B receives a new epoch and A's
checkpoint. B obtains focused context with:

```sh
stackcord resume --project shop --task api --max-bytes 16384 --token-file /private/path/token
```

B completes with its own returned tuple. A cannot complete with the old tuple.
If A returns later, it reads current context and claims only eligible work.
If a worker dies before checkpointing, only its last persisted progress is
recoverable. A lease expiry permits reassignment but does not stop the old process.

A pause reason of `approval_required` cannot be automatically claimed. An explicit
`requeue` of a paused/failed task requires the current `expected_revision`:

```json
{"id":"retry-api","project":"shop","task":"api","kind":"requeue","expected_revision":3}
```

Requeue preserves the checkpoint. It is a trusted operator action, not proof of
product approval. Inspect partial changes and required approvals before using it.
The new runtime does not itself launch processes or infer provider quota/reset times.

## Changes and context budgets

```sh
stackcord projects --token-file /private/path/token
stackcord task list --project shop --token-file /private/path/token
stackcord events --project shop --after 0 --limit 100 --token-file /private/path/token
```

Persist the returned `next` cursor only after consuming the page. Continue while
`more` is true. Cursors are project-specific and scoped to this database. A cursor
ahead of a restored backup fails explicitly; reconcile the restore before resetting
it. Do not silently discard unprocessed events.

Events contain identity/state changes, not repeated goals or logs. Use `resume`
only for the task you need. Large artifacts stay at referenced locations; the
runtime never fetches them. Mandatory constraints are never silently truncated.
An insufficient byte budget returns an error with the required size. Bytes are
not tokenizer-exact token counts. Usage totals contain only reported increments,
counted once per mutation ID. No token savings benchmark is implied by byte counts.

## Remote workers and data safety

For multiple computers, run one coordinator on a reachable machine and connect
using an authenticated SSH tunnel (loopback HTTP), or host the library handler
behind an operator-managed HTTPS service. The shipped CLI exposes no LAN bind flag.
The HTTP client refuses non-loopback plain HTTP and redirects. Requests require
bearer auth; cross-origin browser requests are rejected.

One credential authorizes all projects and operator actions in that runtime.
Worker names are not authentication principals. Separate untrusted tenants.
The database is not encrypted by Stackcord; protect its directory with OS access
controls. Do not put it in Git or on a network filesystem. Recorded tasks, goals,
checkpoints and result references may contain private project information.

## Backup, restore and existing Git channels

```sh
stackcord backup --output /private/backup/stackcord.db --token-file /private/path/token
```

Backup refuses to overwrite an existing file and removes an incomplete output on
failure. It is a consistent database snapshot. To restore: stop the runtime, retain
the previous database, copy the backup to a new private local path, then restart
`serve --db` with that path and your credential. Git clone alone cannot restore
runtime progress. No automatic retention deletion or disk-space compaction runs.

Legacy `stackcord channel` remains under the optional harness, with its signed Git
history, pinned peers and local execution receipts intact. There is no implicit
migration between its state and this runtime. Finish or pause existing work there;
start new runtime tasks with references to relevant Git commits and decisions.
The original command paths remain hidden aliases for installed Skills. Prefer
`stackcord harness channel`, `stackcord harness status` and
`stackcord harness dashboard` for explicit extension use.

The bundled web console supports loopback and SSH tunnels. An HTTPS API reverse
proxy is not a supported console deployment: browser origins must match the
handler’s explicit host and TLS scheme. No forwarded headers are trusted.
Definitions are bounded to 32 KiB, checkpoints to 24 KiB, and aggregate resume
context to 1 MiB (default 16 KiB). Larger inputs belong behind artifact references.
