# port-mapper

A small local UPnP port-mapping tool.

## What it is

- Go backend
- Svelte + Vite frontend
- local-only web UI on `127.0.0.1`
- `config.json` stored beside the binary
- frontend embedded into the Go binary

## Quick start

```bash
cd backend
go run ./cmd/port-mapper
```

The app will:

- load `config.json` from the same directory as the executable
- fall back to safe defaults when the file is missing
- bind only to `127.0.0.1`
- try to open your browser automatically

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
