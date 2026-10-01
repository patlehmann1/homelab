# Hearth

Home lab / NAS dashboard. It sits in front of Immich (photos), Paperless-ngx (documents), Jellyfin (media), Pi-hole (DNS) and restic/Backrest (backups), and adds storage health, services and family accounts.

Everything currently runs on in-memory mock data (`seed.go`), since the hardware doesn't exist yet.

```sh
go run .            # http://localhost:8080
PORT=9000 go run .
go test ./...
```

## Layout

- `main.go`: routes, handlers, template funcs
- `store.go`: data types and the in-memory store. `advance()` fakes live metrics.
- `seed.go`: mock data
- `templates/layout.html`: page shell, top nav (a bottom dock on phones), and SVG icons
- `templates/components.html`: fragments that htmx swaps in on their own
- `templates/pages/*.html`: one file per section
- `static/`: CSS and vendored htmx 2.0.4

## How the htmx parts work

The server only ever returns HTML. htmx attributes say when to fetch it and where to put it.

| Pattern | Where | Attributes |
|---|---|---|
| SPA-style nav | top nav | `hx-boost` on the nav. The server sees `HX-Target: main`, returns only `<main>` content, and re-sends the nav with `hx-swap-oob` so the active tab updates. |
| Polling | vitals, Immich jobs, DNS stats | `hx-trigger="every 3s"` with `hx-swap="outerHTML"`. Each response includes the trigger again, so polling continues. Meters keep stable ids so htmx's settle step animates their width. |
| Poll until done | service restart, backup run | The row only includes `hx-trigger="every 500ms"` while it is `running` or `restarting`. Once it's done, the returned row has no trigger and polling stops. |
| Active search | documents | The form uses `hx-trigger="input delay:250ms"` and swaps `#doc-results`. |
| Append / remove | family | `hx-post` with `hx-swap="beforeend"` adds a person card. `hx-delete` with `hx-target="closest .person"` removes one. |
| Out-of-band | Pi-hole toggle | The response includes the toggle and a fresh nav, so the Network tab's status dot changes too. |

## Next, when the hardware arrives

Replace `seed.go` with real clients for each one: the Immich API (`/api/server/statistics`), Paperless (`/api/documents/?query=`), the Jellyfin Sessions API, the Pi-hole API, the Docker socket, and `smartctl`/`zpool status` for the disks.
