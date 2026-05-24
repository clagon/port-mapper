# Usage

## What this app does

`port-mapper` is a local-only UPnP port mapping tool.
It runs a Go backend on `127.0.0.1`, serves the Svelte UI, and stores `config.json` beside the binary.

## Quick start

```bash
cd backend
go run ./cmd/port-mapper
```

On startup, the app:

- loads `config.json` from the same directory as the binary
- falls back to safe defaults when the file does not exist
- binds only to `127.0.0.1`
- opens the local UI in your browser when browser opening is enabled

## Config file

The runtime config lives next to the executable:

```text
/path/to/port-mapper
/path/to/config.json
```

If `config.json` is missing, the app uses defaults.

### Example

```json
{
  "listen_addr": "127.0.0.1:8080",
  "auto_discover": true
}
```

## UI workflow

1. Start the app.
2. Open `http://127.0.0.1:8080` if the browser does not open automatically.
3. Let discovery run.
4. Re-run discovery manually if your router was not detected on first start.
5. Open or close mappings from the UI.

## Notes

- The server is intentionally local-only.
- If you need to change the port, edit `config.json` or pass a different listen address through startup wiring.
- For IP/port validation and lease behavior, see `docs/security.md`.
