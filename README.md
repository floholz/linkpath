# LinkPath

A URL-path-aware link and note aggregator. Prepend your LinkPath domain to any URL path to get a personal page for storing links and notes scoped to that path.

```
linkpa.th/github.com/myorg/project  →  your links & notes about that repo
linkpa.th/github.com/myorg          →  your links & notes about the org
linkpa.th/github.com                →  your links & notes about GitHub
```

Paths are hierarchical — parent path items are shown in collapsible sections below the current path, so context is always a scroll away.

## Features

- **Path-organized** — any URL path becomes a personal page
- **Links & Notes** — save links with optional display text, or write Markdown notes
- **Hierarchy** — sidebar shows parent and child paths; parent items shown inline for context
- **Manual ordering** — drag items up/down to reorder within a path
- **Private** — all items are per-user, nothing is public
- **Single binary** — templates and static files embedded; one container, no external services

## Tech Stack

| | |
|---|---|
| **Go** | HTTP server, all business logic |
| **PocketBase** (embedded, v0.23+) | SQLite database, user auth, admin UI |
| **HTMX** | Dynamic frontend interactions (CDN) |
| **goldmark** | Server-side Markdown → HTML rendering |

## Running Locally

```bash
go run . serve
```

| | Default |
|---|---|
| App | `http://localhost:8080` |
| PocketBase admin | `http://localhost:8090/_/` |

Data is persisted in `./pb_data/`.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `APP_HTTP` | `0.0.0.0:8080` | App server address |
| `PB_HTTP` | `0.0.0.0:8090` | PocketBase server address |
| `APP_HOST` | `linkpa.th` | Hostname shown on the landing page |

These can also be passed as CLI flags: `--app-http`, `--http`.

## Deployment

Single Docker container, intended to run behind Traefik for TLS termination.

```bash
docker compose up -d
```

The `docker-compose.yml` includes Traefik labels for both the app (`linkpa.th`) and the PocketBase admin UI (`pb.linkpa.th`). Edit the hostnames and env vars to match your setup.

`pb_data/` is mounted as a volume — database survives container restarts and rebuilds.

### Build

```bash
docker build -t linkpath .
```

CGO is required (PocketBase uses SQLite). The Dockerfile uses a two-stage build: Go builder on Alpine, minimal Alpine runtime image.

## Project Structure

```
linkpath/
├── main.go                    # entry point, route registration, two-server setup
├── embed.go                   # //go:embed for templates + static files
├── migrations/
│   └── 001_collections.go     # nodes + items schema
├── internal/
│   ├── handlers/
│   │   ├── path.go            # catch-all GET handler (landing, dashboard, path views)
│   │   ├── items.go           # CRUD + reorder HTMX endpoints
│   │   ├── auth.go            # login, register, logout
│   │   └── types.go           # ItemCard, AncestorGroupData helpers
│   ├── middleware/
│   │   └── auth.go            # cookie-based auth middleware
│   ├── pathutil/
│   │   └── normalize.go       # path normalization
│   └── render/
│       ├── templates.go       # per-page template sets
│       └── markdown.go        # goldmark renderer
├── templates/
│   ├── base.html
│   ├── landing.html
│   ├── login.html
│   ├── register.html
│   ├── path.html              # main view: sidebar + items
│   └── partials/
│       ├── item_card.html
│       ├── items_list.html
│       ├── add_form.html
│       └── edit_form.html
├── static/css/
│   └── main.css
├── Dockerfile
└── docker-compose.yml
```

## Routes

All internal app routes are prefixed with `/~/` to avoid conflicting with user path space.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/` | optional | Landing page (unauthed) or dashboard |
| `GET` | `/~/login` | — | Login page |
| `POST` | `/~/login` | — | Authenticate, set cookie |
| `GET` | `/~/register` | — | Register page |
| `POST` | `/~/register` | — | Create account, set cookie |
| `POST` | `/~/logout` | ✓ | Clear cookie |
| `GET` | `/~/items/add-form` | ✓ | Add item form partial |
| `POST` | `/~/items` | ✓ | Create item |
| `GET` | `/~/items/{id}` | ✓ | Item card partial |
| `GET` | `/~/items/{id}/edit` | ✓ | Edit form partial |
| `PUT` | `/~/items/{id}` | ✓ | Update item |
| `DELETE` | `/~/items/{id}` | ✓ | Delete item |
| `POST` | `/~/items/{id}/move` | ✓ | Reorder item (`?direction=up\|down`) |
| `GET` | `/{path...}` | ✓ | Path view |

## Architecture Notes

**Two servers** — PocketBase runs on port 8090 (internal, admin only). The app runs on port 8080 (public, behind Traefik). This separation avoids route conflicts between PocketBase's own API routes and the catch-all path handler.

**Per-page template sets** — each page gets its own `*template.Template` instance to prevent `{{define "content"}}` blocks from overwriting each other across pages (a Go `html/template` gotcha).

**Path normalization** — the first path segment (domain) is lowercased; subsequent segments are case-preserved. Requests that differ from their normalized form receive a `301` redirect.
