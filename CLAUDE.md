# LinkPath

A URL-path-aware link and note aggregator. Inspired by Linktree but uses the existing URL path structure as the organizational hierarchy.

## Concept

Prepend the app domain to any URL path to get a personal page for storing links and notes scoped to that path. Paths are hierarchical — `/fhfun.cc/games/stdlible` is a child of `/fhfun.cc/games`, which is a child of `/fhfun.cc`.

## Tech Stack

- **Go** — HTTP server, all business logic
- **PocketBase** (embedded, v0.23+) — SQLite DB, user auth, admin UI at `/_/`
- **HTMX** — dynamic frontend interactions (loaded from CDN)
- **Go `html/template`** — server-side HTML rendering
- **goldmark** — server-side Markdown → HTML for notes
- Templates and static files are **embedded in the binary** via `//go:embed`

## Running

```bash
go run . serve --http=0.0.0.0:8090
```

PocketBase admin UI: `http://localhost:8090/_/`
App: `http://localhost:8090/`

Data is persisted in `./pb_data/` (mounted as a Docker volume in production).

## Project Structure

```
linkpath/
├── main.go                    # PocketBase init, OnServe hook, route registration
├── embed.go                   # //go:embed directives for templates + static
├── migrations/
│   └── 001_collections.go     # nodes + items collection definitions
├── internal/
│   ├── pathutil/
│   │   └── normalize.go       # path normalization logic
│   ├── middleware/
│   │   └── auth.go            # cookie-based auth middleware
│   ├── render/
│   │   ├── templates.go       # template loading and execution helpers
│   │   └── markdown.go        # goldmark renderer
│   └── handlers/
│       ├── auth.go            # /login, /register, /logout
│       ├── path.go            # GET /{path...}
│       └── items.go           # CRUD for items (HTMX endpoints)
├── templates/
│   ├── base.html
│   ├── landing.html           # shown to unauthenticated users
│   ├── login.html
│   ├── register.html
│   ├── path.html              # main view: sidebar tree + items
│   └── partials/              # htmx swap targets
│       ├── item_card.html
│       ├── add_form.html
│       └── edit_form.html
├── static/css/
│   └── main.css
├── Dockerfile
└── docker-compose.yml
```

## Database Collections

### `nodes`
Represents a unique normalized path. Created on first visit to a path.

| Field  | Type | Notes                  |
|--------|------|------------------------|
| `path` | text | required, unique, indexed |

### `items`
Links or notes attached to a node, owned by a user.

| Field        | Type                  | Notes                        |
|--------------|-----------------------|------------------------------|
| `node`       | relation → nodes      | required                     |
| `user`       | relation → _pb_users_ | required                     |
| `type`       | select: link\|note    | required                     |
| `title`      | text                  | optional, max 200            |
| `url`        | url                   | optional, links only         |
| `body`       | text                  | optional, notes only (markdown) |
| `sort_order` | number                | default 0                    |

Both collections have API rules locked down — all access goes through Go handlers.

## Path Normalization

The domain segment (first path component) is lowercased; subsequent segments are kept as-is.

```
FHFun.CC             → fhfun.cc
FHFun.CC/Games       → fhfun.cc/Games
FHFun.CC/Games/Foo   → fhfun.cc/Games/Foo
```

If a request path differs from its normalized form, the handler issues a `301` redirect.

## Auth

- Auth token stored in an HTTP-only `pb_auth` cookie, set by the Go server after login
- `AuthMiddleware` validates the cookie on every protected route using `app.FindAuthRecordByToken()`
- All items are **private per user** — users only see their own links and notes
- Unauthenticated users see a landing/info page and cannot access any path pages

## Routes

| Method   | Path              | Auth | Description                          |
|----------|-------------------|------|--------------------------------------|
| GET      | `/`               | opt  | Landing (unauthed) or dashboard      |
| GET      | `/login`          | no   | Login page                           |
| POST     | `/login`          | no   | Process login, set cookie            |
| GET      | `/register`       | no   | Register page                        |
| POST     | `/register`       | no   | Create user, set cookie              |
| POST     | `/logout`         | yes  | Clear cookie                         |
| GET      | `/{path...}`      | yes  | Path view (sidebar + items)          |
| POST     | `/items`          | yes  | Create item — returns HTMX partial   |
| GET      | `/items/{id}`     | yes  | Item card partial (used for cancel)  |
| GET      | `/items/{id}/edit`| yes  | Edit form partial                    |
| PUT      | `/items/{id}`     | yes  | Update item — returns HTMX partial   |
| DELETE   | `/items/{id}`     | yes  | Delete item — returns empty          |

## Path View Logic

`GET /{path...}` does the following:

1. Normalize path; redirect if changed
2. Find-or-create `node` for this path
3. Load current user's items at this node
4. Compute ancestor paths and load their items (for context display)
5. Load all descendant nodes to build the sidebar tree
6. Render `path.html`

Ancestor items are shown below current items, grouped by path (nearest parent first), in collapsible sections.

## HTMX Patterns

- **Add item**: inline form loaded via `hx-get`, submitted via `hx-post` → new `item_card.html` prepended to list
- **Delete**: `hx-delete` with `hx-swap="outerHTML"` on the card → empty response removes element
- **Edit**: `hx-get` loads edit form replacing the card; save/cancel swap back to card

## Deployment

Single Docker container, runs behind Traefik (which handles TLS and host routing).

```bash
docker compose up -d
```

`pb_data/` is mounted as a volume for database persistence. No external services required.
