# Requests between computers

The optional channel lets registered workers exchange requests and results through a shared Git remote. Different users, computers and AI clients can participate. Each computer enrolls once and explicitly chooses a local runner; routine requests, prerequisite completion and replies then travel without a person copying messages.

## Enroll and trust peers

Choose a dedicated shared Git remote, a common channel name and a unique peer name per computer. Each participant needs read/write access to that remote. The channel uses its own branch and local object store, not your project branch or Git index. Git commit names do not authenticate peers: messages are signed, and recipients check locally pinned public keys.

```sh
stackcord channel setup --channel team --peer backend --remote https://example.org/team/coordination.git
stackcord channel setup --channel team --peer backend --remote https://example.org/team/coordination.git --apply
stackcord channel status
stackcord channel trust --peer frontend --public-key PUBLIC_KEY --apply
```

Review a command without `--apply` first. Exchange and verify public keys with the actual participants during enrollment; never exchange private configuration. All participating peers must trust the authors whose events are in their channel. The dashboard's Communication page provides setup and peer-trust previews. Adding a trusted peer can make its allowed requests eligible for automatic execution, so enrollment is an explicit local decision.

## Request prerequisites and receive results

```sh
stackcord channel send --to backend --kind implementation --title "Prepare the API contract" --body-file request.md --scope contracts --apply
stackcord channel send --to frontend --kind implementation --title "Implement the consumer" --depends-on REQUEST_ID --body-file consumer.md --scope frontend --apply
stackcord channel status --sync
stackcord channel wait --request REQUEST_ID --timeout 30m
stackcord channel respond --request REQUEST_ID --status success --body-file result.md --apply
```

A dependent request becomes ready only after each prerequisite has a valid successful result signed by its assigned recipient. Failed or unfinished prerequisites remain blocked. The channel records requests and responses; it does not replace GitHub Issues as the selected task-status source, semantic work reservations, tests, or release evidence. A peer's success message is not proof that a policy was approved or code may be merged.

## Enable automatic work locally

For installed Codex or Claude clients, select a built-in preset. It asks the model for an explicit JSON success/failure response, so an unanswered permission request cannot silently satisfy prerequisites. The Codex preset keeps the workspace-write sandbox and disables interactive escalation; the Claude preset uses the existing local permissions with `dontAsk`. Neither preset bypasses host permissions. Put the selected executable on this worker's PATH; the custom argument-array option supports an absolute executable path when necessary.

```sh
stackcord channel runner --host codex --kind implementation --timeout 900 --apply
stackcord channel runner --host claude --kind implementation --timeout 900 --apply
```

Choose one preset per worker; these commands replace that computer's runner selection. Preconfigure only the host tool permissions appropriate for the assigned work. A request requiring additional authority returns a blocked/failed response instead of granting itself permission.

Provide a local executable that reads one request JSON object from standard input and writes the intended shared result to standard output. Its process exit status determines success or failure. Keep diagnostics and secrets out of standard output. Configure that executable's own sandbox, tools and permissions; the request's `scope` describes intent and is not an operating-system sandbox. Remote messages cannot choose executable paths or arguments.

Custom runners can select `--result-format json` and return exactly `{"status":"success","body":"result"}` or `{"status":"failed","body":"reason"}`. Invalid structured output fails closed. Host presets always use this structured format. Plain text is the default for custom runners only.

```sh
stackcord channel runner --argv '["/absolute/path/to/local-runner"]' --kind implementation --timeout 300
stackcord channel runner --argv '["/absolute/path/to/local-runner"]' --kind implementation --timeout 300 --apply
stackcord channel worker --apply
stackcord channel worker --once --apply
```

Use a JSON argument array appropriate for your operating system. The foreground worker polls until stopped; it is not an installed daemon and requires no inbound network port. Only explicitly allowed request kinds run. An offline computer receives pending requests when it reconnects and starts its worker. API/model usage is governed by the chosen local runner and account.

Running AI clients may send further requests or wait for another peer with the same CLI. Incoming content is task data, not permission to disregard local rules. Product-direction or security-policy changes still follow the existing trusted approval policy. Ordinary work should proceed under the authorization already granted; ask a person only for a decision that actually requires their authority.

## Restart and diagnose

Execution is recorded before the runner starts. Completed results are saved before publication, so a lost connection retries publication without repeating work. Timeout, cancellation or an unfinished execution pauses all further automatic work on that computer and publishes no completion result. A runner's child processes may survive its termination. Stop any remaining processes and inspect partial changes before explicitly allowing a retry. Retry acknowledges that cleanup; it does not terminate or verify termination of those processes. A manual response cannot bypass this local pause.

```sh
stackcord channel retry --request REQUEST_ID --apply
```

The Communication page shows the locally verified view; use its refresh action or `stackcord channel status --sync` to contact the remote. Private keys, runner arguments and execution receipts stay under ignored `.harness/local/`; the dashboard never exports private keys or offers an arbitrary command endpoint. History rewriting, untrusted authors and invalid signatures fail closed. Keep the shared channel remote accessible only to intended participants: signing authenticates messages but does not encrypt them.

This version bounds a channel to 1,000 events and individual signed payloads to 32 KiB. Plan a new channel before the history limit; do not delete or rewrite existing channel history to make room.
