# Hosting a private packing trip

Hosted mode serves one durable private trip independently of the browser or
original data checkout. It saves the exact draft, accepted list, and packing
progress. Access uses a private trip link with edit/pack permissions and operator
recovery. Cross-device updates use HTTP: another open browser catches up on
Refresh trip, page opening/reload, visibility return, or connectivity return.
Live updates through SSE are not implemented.

## Setup

Follow the [README build quick start](../README.md#quick-start), or rebuild with:

```sh
make frontend
go build -o /tmp/packing-server ./cmd/packing-server
```

The commands below assume `packing-server` is installed on your PATH; substitute
`/tmp/packing-server` when using the build above.

Use a private persistent local filesystem directory, owned by the service user.
The Linux implementation has one active writer and refuses a second process.
The store must not live in an ephemeral container layer or on a network/shared
filesystem. Run the service as a non-root user. Keep stores, access-link files,
backups and private source definitions outside the repository and public images.

Provision a reviewed, flat definition bundle (at most 512 files, 2 MiB total).
Only regular files with keys matching `[A-Za-z0-9][A-Za-z0-9_.-]{0,127}` are
accepted; symlinks, traversal names, missing references and invalid definitions
are refused. Copy only definitions intended to be available to this trip's link
holders. `ref:` resolution cannot reach the original library outside this copy.

```sh
packing-server hosted --store /srv/packing/trip init \
  --groups-dir /private/reviewed-trip-groups \
  --selection-file /private/chosen-trip
packing-server hosted --store /srv/packing/trip access \
  --origin https://packing.example \
  --output /private/packing-access-link
packing-server hosted --store /srv/packing/trip serve \
  --origin https://packing.example --listen 127.0.0.1:3865
```

`init` is create-only and repeat-safe: it refuses an initialized store. It copies
and pins definitions without changing the originals. The optional selection
file becomes an exact, unapplied draft; even invalid draft syntax is retained
for repair. The accepted list starts empty. Open the private link, review the
preview and Apply explicitly. No existing browser progress is inferred or imported.

Put a TLS reverse proxy on the same host in front of the loopback listener.
The public origin must exactly match `--origin`, including a nondefault port.
For example, after choosing and configuring a real domain, Caddy can terminate
HTTPS and proxy to `127.0.0.1:3865`:

```caddyfile
packing.example {
    reverse_proxy 127.0.0.1:3865
}
```

This is a deployment template, not a deployed service. Preserve the incoming
Origin header, disable caching of application responses, and do not log request
bodies or cookies. The application supplies no-store, CSP, nosniff, no-referrer
and HSTS headers. Protect the service from direct public HTTP access. The
explicit `--insecure-localhost` flag permits only numeric loopback HTTP origins
and loopback listeners for synthetic tests; never use it for hosted service.

`/health/live` and `/health/ready` expose no trip contents. Header, body, request
and write timeouts are bounded. HTTP mutation/resolution concurrency is capped
at eight; access exchanges are capped at twenty per minute per process.

## Access and recovery

Anyone holding the secret link can read, edit, resolve the pinned definitions,
Apply, and check/uncheck this one trip. The link does not permit creating trips,
issuing links, operating backups or reading an unrelated library. Sharing it
shares full trip edit/pack access; separate guest roles are deferred.

The fragment secret is exchanged in a same-origin POST for a 30-day, Secure,
HttpOnly, SameSite=Strict browser cookie. The fragment is removed after exchange.
The cleaned stable URL resumes an authorized browser but does not authorize a
new device: transfer the original secret link privately. Do not put it in an
issue, repository, screenshot, analytics service or URL shortener.

Only hashes of random capability and browser tokens are stored. The capability
lasts until rotated; cookie expiry requires opening the original link again.
At most 100 unexpired browser sessions are retained. There is no hosted login
provider or public registration. Browsers expose no access-administration API.

To revoke lost/forwarded links or recover owner access, stop the service and run
`hosted access` again with a **new output file**. It revokes all previous links
and browser cookies without changing the trip. The new secret is written once
to that private file, never printed. An output failure is recoverable by issuing
another link. Keep host/operator access and the link in appropriately private
storage. Closing a tab is not logout; revoke access or clear site data on a
shared device after preserving any unsaved work.

## Editing, conflicts and device recovery

- Draft edits autosave after a short debounce, including incomplete/invalid text.
  Saving a draft does not Apply it or alter the accepted list/progress.
- Apply requires a saved, reviewed draft and the current snapshot generation.
  Surviving exact item names keep their packed state. New or reintroduced items
  start unpacked. The same parsed item name in several groups shares progress,
  preserving the existing parser and UI semantics.
- Each item has its own conditional version. Different devices can change
  different items without losing either change. Stale writes to the same item
  or draft are refused; an old-generation item write cannot modify a new list.
- The UI shows saving, unsynced, conflicting and saved states. Only acknowledged
  changes are promised on another device. Pending edits survive refresh and
  recover from local storage on reload when storage is available.
- Retry uses the original mutation ID, body and preconditions. A lost response
  can therefore recover an already committed result without repeating the edit.
  A stale receipt never replaces a newer confirmed aggregate.
- After a conflict, review the displayed shared draft and accepted list. Either
  discard device edits and use the shared trip, or explicitly save your draft
  over the shown shared draft. The latter still checks for intervening edits.
  For a conflicting item, discard the pending device edit, review the current
  list, and make a fresh choice. There is no automatic whole-set overwrite.
- “Restore draft to accepted list” is a conditional shared-draft edit, not a
  local-only discard. Bulk progress reset is deferred. Packing clicks wait for
  current device writes to finish; draft typing can continue during a save.

Device recovery uses versioned session storage scoped to this tab, origin, access
grant and trip, retaining the dataset epoch and pending request identities. This
prevents tabs from overwriting each other's pending recovery. A fresh page gets a
new mutation-ID prefix; recovered requests retain their original IDs. Storage
failures are shown explicitly; keep the tab open or use **Export device recovery**.
Recovery survives reload, but closing a tab/browser may discard unsynced work;
export it first when needed. No unload/background-save guarantee is made. A fresh access grant gets a fresh
cache scope; export pending work before switching to a rotated link. Exported
recovery is private data, contains no access secret, and is not a complete trip
backup. Preserve an unreadable recovery payload before discarding it.

Mutation receipts are retained for seven days, up to 10,000 entries. Capacity
exhaustion refuses new writes rather than prematurely forgetting retries.
The UI conservatively refuses retries from recovery sessions six days old or
older: refresh/review and explicitly reassert the desired edit. It never
silently gives an old request a new conditional version.

## Persistence, backups and rollback

Hosted mode uses a bounded atomic JSON store. `state.json` includes the immutable
definitions, trip, independent versions, receipts, and hashed access records. An advisory lifetime lock prevents
another server/operator writer. Each commit writes and fsyncs a private temporary
file, atomically renames it, then fsyncs the directory before acknowledgement.
Writes and consistent reads are serialized in-process. After a commit I/O error,
the process fails closed until restarted and reloaded from disk.

Schema 1 and a 32 MiB image cap keep this store limited to one selected trip.
Drafts/individual definitions are limited to 64 KiB and resolved graphs to 10,000
memberships. The JSON image is rewritten per commit; no throughput claim is made.
A later increase in scale can migrate this explicit schema to a database.
The repository remains the source of definitions; no browser library writes or
bidirectional Git synchronization are implemented.

Backups deliberately require a stopped server, taking the same exclusive lock:

```sh
# Stop the service through your service manager first.
packing-server hosted --store /srv/packing/trip backup \
  --output /private/backups/packing-2026-10-15.json
# Copy the private backup to an independently stored, private destination.
# Restart the service after the backup command completes.

# Restore drill into a different, empty persistent directory:
packing-server hosted --store /srv/packing/restore-test restore \
  --input /private/backups/packing-2026-10-15.json
packing-server hosted --store /srv/packing/restore-test access \
  --origin https://restore.example --output /private/restore-access-link
```

Restore never overwrites an initialized store. It retains the exact trip and
pinned definitions, rotates the dataset epoch, clears old receipts and revokes
access. Issue a new private link before serving it. Old clients cannot replay
pre-restore mutations into it. Compare the accepted list, exact draft, item
states and pinned-definition resolution in the independent restored instance.
Choose backup frequency, independent destination and recovery targets before
production reliance; no automatic schedule or approved loss/recovery target is
configured automatically.

For rollback, stop writes, preserve the current store and take a backup first.
An older binary must reject an unsupported store schema. Roll back application
code only to a version supporting the current schema; do not restore an older
backup over later acknowledged edits. Keep those edits for explicit reconciliation.

## Existing browser carryover

A clean new trip requires no browser migration. Existing local-mode progress is
not automatically adopted. If carrying it over later, select the authoritative
browser explicitly. At its original origin, export the correct version-2
localStorage value (`appState` or its selection-file-scoped key), preserving the
accepted snapshot, draft, and done set separately. Keep other devices' exports
separate and retain all original data. Compare the chosen accepted snapshot with
the pinned definitions before Apply, reproduce the chosen packing states in the
hosted trip, then verify both devices. If snapshots differ, resolve that discrepancy
before importing; do not silently substitute current definitions or union sets.
This reviewed one-trip procedure does not implement a general migration UI.

## Verification

```sh
make check
go test -race ./internal/hosted ./cmd/packing-server
```

All fixtures use synthetic private stores. The hosted browser suite uses the
real Go server, two independent Chromium contexts at desktop/mobile sizes,
request barriers, access exchange, and operator backup/restore commands.
It checks cross-device draft/Apply/progress, conflicting writes, delayed old
snapshot writes, lost-response reload/retry, resume refresh and invalid drafts.
Backend tests cover process locking, failures, receipt replay, access revocation,
origin checks, input boundaries and protected legacy routes. Pure Elm tests
cover pending intent, response ordering and recovery codecs.

For each deployment, verify HTTPS and proxy behavior at the chosen host/domain.
Use the intended trip and check an acknowledged desktop-to-phone round trip with
the original machine off. Test actual phone suspension and resumption, then
restore an independently stored backup into a separate instance and compare the
draft, accepted list and packing progress. Desktop mobile emulation does not
verify actual phone or hosted deployment behavior.
