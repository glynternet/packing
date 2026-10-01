# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`packing` generates packing lists from plain-text "group" definition files. It is a Go
monorepo with two binaries plus an Elm single-page app:

- **packing-server** (`cmd/packing-server`) — HTTP server. Serves the Elm SPA and a
  `POST /groups/` JSON endpoint that loads group files from a directory, recursively
  resolves references, and returns the resolved groups.
- **packing-cli** (`cmd/packing-cli`) — CLI client. Reads a local "selection" file,
  POSTs it to the server's `/groups/` endpoint, and renders the response
  (html/markdown/json/item-list).

The server and CLI never share process state; they communicate over HTTP using the shared
JSON types in `pkg/api/api.go` (`Contents{Refs, Items, Requires}` and `Group{Name, Contents}`).

## Commands

Build and test are driven by Make (including dubplate-generated Makefiles), Go,
and project-local frontend tools. Development prerequisites: Go 1.25+, Node.js
24 (the tested version), npm, and Make. Elm, elm-format, elm-test, and Playwright
are pinned in `frontend/package.json` and `frontend/package-lock.json`.

```sh
make binaries            # build both binaries into ./build/<version>/<os>-<arch>/
make binary APP_NAME=packing-server   # build a single binary
make installs            # go install both binaries (version-stamped) to $GOPATH/bin
make setup-frontend      # npm ci + download Playwright Chromium (first setup)
make frontend            # elm make src/Main.elm -> cmd/packing-server/elm.js

make test                # Go tests + Elm unit/fuzz tests
make test-browser        # build UI/server + isolated Chromium acceptance tests
make check               # Elm formatting + all of the above tests

go test ./...                                 # Go tests only
go test ./pkg/list/...                        # test a single package
go test ./pkg/list/ -run TestParseContentsDefinition   # run a single test
go build ./...                                # quick compile check

# run locally
go run ./cmd/packing-server serve --groups-dir <dir> --port 3865
go run ./cmd/packing-server serve --groups-dir <dir> --selection-file <trip-file>
go run ./cmd/packing-cli selection <selection-file> --renderer markdown
```

Notes:
- `serve --selection-file <path>` enables a single shared selection text file.
  The UI loads it on startup, creating an empty file if missing. Apply writes a
  temporary file in the same directory, syncs it, then atomically renames it over
  the selection, preserving existing permissions (new files use `0600`). Use a
  regular file, not a symlink; its parent directory must exist and be writable.
  `GET /saved-selection/` reads/creates the file; `PUT /saved-selection/` validates
  syntax and saves the exact UTF-8 text (up to 1 MiB). GET and successful PUT
  return a strong content-hash `ETag`. PUT requires that exact ETag in `If-Match`:
  missing headers return 428 and stale versions return 412 without writing.
  Reads and compare-and-save operations share a lock within one server instance.
  Edits made outside the server before the comparison are detected, but external
  writers and other server processes do not participate in this lock.
- With this flag, Apply accepts the preview only after the server save succeeds.
  Load/save errors are shown with retry controls. Drafts, resolved snapshots and
  packing progress remain local, scoped to the absolute selection-file path.
  A conflict loads and resolves the latest server selection as Current, retains
  the draft as Proposed, and opens Changes for review. Apply and Discard stay
  blocked until the newer baseline resolves; failures offer a retry. Applying
  again uses the new ETag and can conflict again. No automatic merge is performed.
  Reload retains the local snapshot and draft when its accepted text matches the
  server file; changed server text with an unapplied draft starts conflict review.
  Without a draft it loads the server selection. Without the flag, the existing
  browser-only storage behavior is unchanged.
- Build artifacts go to `./build/` (gitignored). When building only to verify, prefer
  building into `/tmp` so nothing lands in the tree.
- Go tests use `stretchr/testify` and live in external `_test` packages. There is
  no CI configuration; `make check` is the repeatable local validation workflow.

### Frontend testing workflow

`make setup-frontend` installs the locked dependencies and Chromium. Normal builds
use `frontend/node_modules/.bin/elm` (compiler 0.19.2), so the global Elm version
does not affect this project. If only building or running Elm tests, browser
installation is optional: `npm --prefix frontend ci` is sufficient. Run setup
again when updating the Playwright version. Linux environments missing browser
shared libraries can use `cd frontend && npm exec -- playwright install --with-deps chromium`;
that command may require administrator permissions to install OS packages.

Use the narrowest relevant checks while iterating, then `make check` before
finishing a frontend change:

```sh
make -C frontend test          # pure state, diff, persistence, and Elm integration tests
make -C frontend check-format # validate Elm source/test formatting
make -C frontend format       # apply the pinned formatter
make test-browser             # real UI with the real Go parser/resolver

# Forward elm-test arguments, e.g. to reproduce the seed printed by a failure:
npm --prefix frontend test -- --seed 12345 --fuzz 100

# Optional screenshots of the desktop and mobile review flow:
PACKING_TEST_ARTIFACTS=/tmp/packing-screenshots make test-browser
```

The browser suite in `frontend/tests/browser/workflow.mjs` rebuilds the tracked
embedded `cmd/packing-server/elm.js`, compiles a server binary into a unique OS
temporary directory, creates small group fixtures, and starts it on a free local
port. Each test uses a fresh Chromium context and fixture localStorage. It does
not need a running development server or real packing data. The server, browser,
and temporary fixtures are cleaned up after the run; requested screenshots remain
in `PACKING_TEST_ARTIFACTS`. On an abruptly killed process, temporary directories
can remain with the prefix `packing-browser-`.

Unit/fuzz tests cover shared items and source changes, exact Apply/Discard
snapshots, delayed resolution and Simplify responses, zero-item-diff edits,
migration, invalid saved drafts, and packing-progress isolation. Browser tests
exercise group swaps, preview/list/graph switching, Apply without refetching,
invalid-draft reload recovery, delayed responses, Simplify, empty selections,
changed group files, storage failures, responsive layout, and the editor filling
the available pane height before scrolling its contents. Delayed-response
tests use explicit request gates rather than fixed sleeps. Browser runtime errors
fail tests. Group-item tests cover recursive flattening, shared descendants and
cycles, navigation without fetching, completion filters, snapshot isolation, and
mobile layout. Add pure behavior tests to `frontend/tests/*.elm` and user workflow
tests to `frontend/tests/browser/*.mjs`.

## Architecture

### Group-file grammar and parsing
A group file is line-based plain text, parsed by `list.ParseContentsDefinition`
(`pkg/list/contents.go`). Everything after `#` on a line is stripped as a comment, then
each line runs through a `ProcessorGroup` chain (`pkg/list/stringprocessor.go`):
1. empty-line skip,
2. `TaggedLineParser` — `ref: <name>` becomes a reference, `req: <name>` a requirement
   (any other `tag:` prefix is an error),
3. otherwise the line is an item name.
Prefix/parse helpers live in `pkg/parse`.

### Server request pipeline (`cmd/packing-server/cmdtree.go`)
`file.ReadCloserGetter(dir)` (`pkg/storage/file`) opens a file per reference name →
`storage.ContentsDefinitionGetter` (`pkg/storage/groupgetter.go`) parses it into
`api.Contents` → `load.Loader` (`internal/load/group.go`) recursively loads every
referenced group (`recursiveLoad`, guarding against self-references via
`SelfReferenceError`) and wraps loose seed items into an "Individual Items" group →
`service.GroupsService` (`internal/service`) is the thin service layer the HTTP handler
calls.

### CLI pipeline (`cmd/packing-cli/cmdtree.go`)
`selection` command: read selection file → `client.GetGroups` POSTs the seed to the
server → `graph.From` (`pkg/graph`) computes the reverse "ImportedBy" import graph → a
renderer writes output. Renderers are built in `rendererFactory`: `json`, `markdown`,
`html` (renders markdown then converts to HTML via `gomarkdown`), and `item-list`;
`html` is the default. The `reference` command queries refs directly and emits JSON.

### Rendering (`pkg/render/markdown.go`)
`SortedMarkdownRenderer` sorts groups by name and emits markdown. Flags
`IncludeEmptyParentGroups` (groups containing only sub-groups) and `IncludeGroupReferences`
(the "Includes groups"/"Included in" cross-links) control what is shown. Markdown is the
canonical intermediate representation; the HTML renderer is markdown → HTML.

### Frontend (`frontend/`)
Elm 0.19.2 app. `make frontend` compiles it to
`cmd/packing-server/elm.js`, which the server embeds with `//go:embed` and serves at
`/elm.js` alongside an inline `index.html`. **After changing Elm, rerun
`make frontend` and rebuild the server** or the embedded JS is stale. Browser tests
perform both steps automatically.

- `src/Main.elm`: UI, list/graph presentation, and interpretation of selection effects.
  Clicking a list group heading opens its flat descendant item list. This temporary
  view uses the active raw snapshot, shares packing progress and completion filters,
  and resets on review-tab changes, Apply, or Discard. Proposed group lists are read-only.
- `src/Selection.elm`: pure draft/accepted state machine. Revision/request tokens
  reject stale resolve and Simplify responses. Apply accepts the matching preview;
  Discard restores the accepted snapshot without fetching. Packing progress is
  kept outside this state machine.
- `src/Packing.elm`: shared group/snapshot types and JSON codecs. Canonical group
  comparisons ignore ordering and repeated memberships. `resolvedItems` follows
  child references with cycle protection and returns unique items sorted by name.
- `src/SelectionDiff.elm` and `src/Review.elm`: unique resolved item additions/removals,
  directly containing source groups, and raw selection line diffs. Diff data is
  computed before singleton collapsing or completed-item filtering.
- `src/StoredState.elm`: version 2 `appState` encoding/decoding. Stores the accepted
  selection and raw resolved snapshot separately from draft text, along with done
  items and display preference. Legacy saved selections become a baseline only
  after successful resolution; migration preserves any newer draft edits.
- `src/State.elm`: localStorage ports. The JS bridge reports write failures back
  to Elm. Reload resolves the draft while retaining the accepted snapshot, making
  changes to group files reviewable rather than silently applying them.

## Conventions and gotchas

- **Generated main.go**: `cmd/*/main.go` are dubplate boilerplate marked
  `Code generated ... DO NOT EDIT`. Put command wiring in each command's `cmdtree.go`
  inside `buildCmdTree`.
- **Config**: cobra flags only — there is no env-var or config-file support. Each command that
  talks to the server owns a `serverAddr` holding its own `--server-host`/`--server-port`, so
  commands cannot read each other's values. Server default port is `3865`.
- **Trailing slash**: `client.GetGroups` posts to `/groups/` (with the slash) on purpose —
  without it the server returns 405 via redirect.
- **proto is unused**: `pkg/api/proto` holds `.proto` files and a `go:generate` directive,
  but generated `*.pb.go` is gitignored and not wired in; the live wire format is the
  hand-written JSON structs in `pkg/api/api.go`.
- **`internal/` vs `pkg/`**: loading/service orchestration is in `internal/`; reusable
  parsing, storage, graph, and render building blocks are in `pkg/`.
