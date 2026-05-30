# Usage

## What this app does

`port-mapper` is a local-only UPnP port mapping tool.
It runs a Go backend on `127.0.0.1`, serves the Svelte UI, and stores `config.json` beside the binary.
It works on Linux, macOS, and Windows.

## Quick start

Requires Go 1.23 or newer.

```bash
cd backend
go run ./cmd/port-mapper
```

You can also tweak startup behavior:

- `--listen-addr 127.0.0.1:9090` to move the local UI to another localhost port
- `--config /path/to/config.json` to point at a different config file beside the binary
- `--no-browser` to disable automatic browser opening

On startup, the app:

- loads `config.json` from the same directory as the binary unless `--config` is set
- falls back to safe defaults when the file does not exist
- binds only to `127.0.0.1` by default
- opens the local UI in your browser unless `--no-browser` is used

## Config file

The runtime config lives next to the executable:

```text
/path/to/port-mapper
/path/to/config.json
```

On Windows, the config sits beside `port-mapper.exe` in the same folder.

If `config.json` is missing, the app uses defaults.

Changes to `listen_addr` and `auto_discover` are persisted to `config.json` and take effect the next time Porto starts.

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
