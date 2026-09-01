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

Build and test are driven by Make (dubplate-generated Makefiles) and plain `go`.

```sh
make binaries            # build both binaries into ./build/<version>/<os>-<arch>/
make binary APP_NAME=packing-server   # build a single binary
make installs            # go install both binaries (version-stamped) to $GOPATH/bin
make frontend            # elm make src/Main.elm -> cmd/packing-server/elm.js

go test ./...                                 # run all tests
go test ./pkg/list/...                        # test a single package
go test ./pkg/list/ -run TestParseContentsDefinition   # run a single test
go build ./...                                # quick compile check

# run locally
go run ./cmd/packing-server serve --groups-dir <dir> --port 3865
go run ./cmd/packing-cli selection <selection-file> --renderer markdown
```

Notes:
- Build artifacts go to `./build/` (gitignored). When building only to verify, prefer
  building into `/tmp` so nothing lands in the tree.
- There is no lint config or CI in the repo; `go test ./...` is the test surface. Tests
  use `stretchr/testify` and live in external `_test` packages.

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
Elm 0.19.1 app (`src/Main.elm`, `src/State.elm`). `make frontend` compiles it to
`cmd/packing-server/elm.js`, which the server embeds with `//go:embed` and serves at
`/elm.js` alongside an inline `index.html`. App state (done items) is persisted to
`localStorage` through Elm ports. **After changing Elm, rerun `make frontend` and rebuild
the server** or the embedded JS is stale.

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
