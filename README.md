# port-mapper

A small local UPnP port-mapping tool for Linux, macOS, and Windows.

## What it is

- Go backend
- Svelte + Vite frontend
- local-only web UI on `127.0.0.1`
- `config.json` stored beside the binary
- frontend embedded into the Go binary

## Quick start

Requires Go 1.23 or newer.

```bash
cd backend
go run ./cmd/port-mapper
```

Useful flags:

- `--listen-addr 127.0.0.1:9090` to use a different local port
- `--config /path/to/config.json` to point at another config file beside the binary
- `--no-browser` to keep the UI closed at startup

The app will:

- load `config.json` from the same directory as the executable unless `--config` is set
- fall back to safe defaults when the file is missing
- bind only to `127.0.0.1` by default
- try to open your browser automatically unless `--no-browser` is used

If the browser does not open, visit:

```text
http://127.0.0.1:8080
```

## Docs

- `docs/usage.md`
- `docs/security.md`

## Backend

The Go backend lives in `backend/` and serves a local HTTP API on `127.0.0.1`.

### Health endpoint

- `GET /api/health` → `{"ok":true}`

### Run

```bash
cd backend
go run ./cmd/port-mapper
```
