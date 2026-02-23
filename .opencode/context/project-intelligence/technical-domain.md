<!-- Context: project-intelligence/technical | Priority: critical | Version: 1.3 | Updated: 2026-02-23 -->

# Technical Domain

> Pressluft is a single-binary Go service that serves an embedded Nuxt 4 dashboard via `embed.FS`.

## Primary Stack

| Layer | Technology | Version | Rationale |
|-------|-----------|---------|-----------|
| Backend | Go | 1.22 | Single-binary deployment, embed.FS for static assets |
| Frontend | Nuxt 4 | ^4.3.1 | Vue 3 meta-framework, static generation for Go embedding |
| UI Framework | Vue 3 | ^3.5.28 | Composition API, reactivity, SFC components |
| CSS | Tailwind CSS v4 | ^4.2.0 | Utility-first, CSS-first config via `@theme {}` blocks |
| Fonts | Inter + JetBrains Mono | Variable | Self-hosted via `@nuxtjs/google-fonts` with `download: true` |
| Database | SQLite | — | Local persistence via `modernc.org/sqlite` (pure Go, no CGo) |
| Migrations | Goose | v3.24.1 | Embedded SQL migrations via `pressly/goose/v3` |
| Cloud SDK | hcloud-go | v2.19.0 | Hetzner Cloud API (pinned for Go 1.22 compat) |
| Build | Make | — | Orchestrates npm generate → copy to embed dir → go build |

## Architecture Pattern

```
Type: Monolith (single binary)
Pattern: Go HTTP server + embedded static Nuxt SPA
Flow: make build → nuxt generate → cp .output/public → internal/server/dist/ → go build → single binary
```

### Why This Architecture?

Single-binary deployment to VPS. No Node runtime needed in production. Go serves the static dashboard and exposes `/api` routes. During development, Nuxt dev server proxies `/api` to the Go backend.

## Project Structure

```
pressluft/
├── cmd/main.go                    # Go entrypoint, DB init, HTTP server
├── internal/
│   ├── database/
│   │   ├── database.go            # SQLite open, pragmas, migration runner
│   │   └── migrations/            # Embedded SQL migrations (goose)
│   │       └── 00001_create_providers.sql
│   ├── provider/
│   │   ├── provider.go            # Provider interface, Info, ValidationResult, registry
│   │   ├── store.go               # StoredProvider type, Store (Create/List/Delete)
│   │   └── hetzner/
│   │       └── hetzner.go         # Hetzner Cloud implementation (hcloud-go)
│   └── server/
│       ├── handler.go             # Route setup, SPA handler, JSON helpers
│       ├── handler_providers.go   # Provider CRUD + validate + types endpoints
│       ├── handler_test.go
│       ├── logging.go             # Request logging middleware
│       ├── logging_test.go
│       └── dist/                  # Embedded static assets (generated, gitkeep)
├── web/                           # Nuxt 4 frontend
│   ├── nuxt.config.ts             # Tailwind v4 vite plugin, Google Fonts, API proxy
│   ├── package.json
│   ├── app/
│   │   ├── app.vue                # Root: <NuxtLayout><NuxtPage /></NuxtLayout>
│   │   ├── assets/css/main.css    # Design system: OKLCH theme, custom utilities
│   │   ├── layouts/default.vue    # Top nav, content area, footer, mobile menu
│   │   ├── composables/           # useModal, useDropdown, useProviders
│   │   ├── components/            # SettingsProviders + ui/ (11 reusable components)
│   │   └── pages/                 # index (dashboard), settings, components (UI library)
│   └── .output/public/            # Generated static output (not committed)
├── Makefile                       # build, dev, run, format, lint, test, check, clean
├── go.mod
└── README.md
```

## Key Commands

| Command | What It Does |
|---------|-------------|
| `make dev` | Starts Go backend (port 8081) + Nuxt dev server (port 8080) with API proxy |
| `make build` | Full pipeline: npm install → nuxt generate → copy to dist → go build → `bin/pressluft` |
| `make run` | Build + run the binary |
| `make check` | format → lint → test → build (full validation) |
| `make test` | Go tests only |
| `make clean` | Remove binary |

## Database Layer

SQLite via `modernc.org/sqlite` (pure Go, no CGo). DB location: `~/.local/share/pressluft/pressluft.db` (XDG-compliant), overridable via `PRESSLUFT_DB` env var.

**Pragmas**: WAL mode, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=NORMAL`, `MaxOpenConns(1)`.

**Migrations**: Embedded SQL files via `pressly/goose/v3`. Files in `internal/database/migrations/`. Use `fs.Sub(embedMigrations, "migrations")` to strip the directory prefix (goose gotcha).

## Provider System

Extensible cloud provider abstraction. Only Hetzner implemented for MVP.

- **Interface**: `Provider` with `Info()` and `Validate(ctx, token)` methods (`internal/provider/provider.go`)
- **Registry**: Global `Register()`/`Get()`/`All()` — providers self-register via `init()` (blank import in `cmd/main.go`)
- **Store**: `provider.Store` wraps `*sql.DB` for CRUD on the `providers` table. API tokens excluded from JSON serialization (`json:"-"`)
- **Hetzner**: Validates via `client.Location.List()` (auth check) + `client.Server.List()` (permission check). Uses `hcloud.IsError(err, hcloud.ErrorCodeUnauthorized)` pattern.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/providers` | List saved providers (tokens excluded) |
| POST | `/api/providers` | Create provider (validates token first) |
| DELETE | `/api/providers/{id}` | Delete provider by ID |
| POST | `/api/providers/validate` | Standalone token validation |
| GET | `/api/providers/types` | List registered provider types |

All endpoints return JSON. Errors use `{"error": "message"}` format. Provider endpoints only registered when DB is available (`db != nil` guard in `NewHandler`).

## Frontend Configuration

### Tailwind CSS v4

- **NOT** using `@nuxtjs/tailwindcss` module (still on Tailwind v3)
- Using `@tailwindcss/vite` as a Vite plugin in `nuxt.config.ts` → `vite.plugins`
- CSS-first configuration: `@import "tailwindcss"` + `@theme {}` blocks in `main.css`
- No `tailwind.config.js` file

### Design System (main.css)

- OKLCH color format for all colors
- Surface scale: 950 (darkest) → 50 (lightest)
- Accent: cyan tones
- Primary: blue tones
- Semantic: success (green), warning (amber), danger (red)
- Custom utilities: `glass`, `glow-accent`, `glow-primary`
- Fonts: `--font-sans: 'Inter'`, `--font-mono: 'JetBrains Mono'`

### Pages (3 routes)

| Route | Page | Status |
|-------|------|--------|
| `/` | Dashboard | Placeholder (headline + subline) |
| `/settings` | Settings | Vertical sidebar sub-nav, 7 sections, query-param routing (`?tab=general`), mobile dropdown fallback. Providers section is functional (add/validate/delete); other sections are placeholder. |
| `/components` | UI Components | Kitchen-sink showcase of all UI components |

### UI Components (11)

UiButton (5 variants, 3 sizes, loading/disabled), UiCard (slots, hoverable), UiBadge (5 variants), UiProgressBar (4 colors, 3 sizes), UiInput, UiSelect, UiTextarea, UiToggle, UiModal (teleported, animated), UiDropdown (click-outside, escape), UiDropdownItem (normal/danger/disabled)

### Feature Components (1)

`SettingsProviders` — Provider management UI: empty state, provider list with status badges, add modal with 2-step flow (validate token → name & save), inline Hetzner tutorial, animated validation feedback (success/warning/error)

### Composables (3)

`useModal()` — open/close/toggle reactive state
`useDropdown()` — click-outside and escape key handling
`useProviders()` — Provider API client (fetchProviders, fetchProviderTypes, validateToken, createProvider, deleteProvider)

### Nuxt Config Highlights

- `css: ['~/assets/css/main.css']`
- `modules: ['@nuxtjs/google-fonts']` with `download: true`, `inject: true`
- `vite.plugins: [tailwindcss()]` from `@tailwindcss/vite`
- `nitro.devProxy: { '/api': { target: 'http://localhost:8081/api' } }`

## Development Environment

```
Requirements: Go 1.22+, Node.js (for Nuxt), npm
Local Dev: make dev (starts both Go + Nuxt with hot reload)
Full Build: make build (produces bin/pressluft)
Testing: make test (Go tests), make check (full validation)
```

## 📂 Codebase References

- `cmd/main.go` - DB initialization, provider registration import, HTTP server wiring
- `internal/database/database.go` - SQLite connection config, pragmas, embedded migrations
- `internal/database/migrations/00001_create_providers.sql` - Providers schema and unique index
- `internal/provider/provider.go` - Provider interface and registry
- `internal/provider/store.go` - Provider persistence layer
- `internal/provider/hetzner/hetzner.go` - Hetzner token validation flow
- `internal/server/handler.go` - Route registration and JSON response helpers
- `internal/server/handler_providers.go` - Provider API endpoints
- `web/app/composables/useProviders.ts` - Frontend provider API client
- `web/app/components/SettingsProviders.vue` - Provider management UI

## Related Files

- `business-domain.md` — Why this project exists
- `decisions-log.md` — Key technical decisions with rationale
- `living-notes.md` — Current state, next steps, gotchas
