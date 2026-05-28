# Security

## Design goals

This tool is intentionally conservative.

- bind only to `127.0.0.1`
- avoid exposing the local API on LAN interfaces
- treat UPnP as a privileged capability and keep the default UX safe
- prefer temporary leases over permanent openings

## Safe defaults

### Local bind only

The backend refuses non-local listen addresses by default.
That means no `0.0.0.0`, no public interfaces, and no accidental LAN exposure.

### Local-only API

The backend binds to localhost by default.
That keeps the UI and API off the LAN unless you explicitly change the listen address.

### Lease duration

Prefer short leases.
Permanent open mappings are risky and should not be the default.

### Dangerous ports

Treat common sensitive ports carefully, especially:

- `22` SSH
- `80` / `443` HTTP(S)
- `3306` MySQL
- `5432` PostgreSQL
- `6379` Redis
- `27017` MongoDB

## Operational advice

- Use the tool only on networks you trust.
- Verify the router and WAN interface before opening a mapping.
- Re-check mappings after router reboots.
- Remove mappings when you no longer need them.

## If something looks wrong

- confirm the app is still bound to `127.0.0.1`
- confirm `config.json` is beside the binary you launched
- confirm the browser token is still valid
- re-run discovery before retrying a mapping
