# Packing

Build packing lists from reusable plain-text groups, review selection changes,
and track packing progress in the Elm frontend.

## Quick start

Building requires Go 1.25+, Node.js 24 (tested), npm and Make. From the repository
root:

```sh
npm --prefix frontend ci
make frontend
go build -o /tmp/packing-server ./cmd/packing-server
/tmp/packing-server --help
```

For a private trip shared across devices, follow the [hosting guide](docs/hosting.md)
to initialize a persistent store, issue a private access link and serve over HTTPS.
It covers setup, access recovery, conflicts, backups, restores and rollback.
Cross-device updates currently use HTTP refresh; SSE live updates are not implemented.

For the local file/browser workflow and development commands, see
[CLAUDE.md](CLAUDE.md).

## Testing

Run `make setup-frontend` once to install frontend tools and the Chromium browser,
then `make check` for the offline Go, Elm and synthetic browser checks.
