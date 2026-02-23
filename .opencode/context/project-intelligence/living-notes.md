<!-- Context: project-intelligence/notes | Priority: high | Version: 1.3 | Updated: 2026-02-23 -->

# Living Notes

> Active state, next steps, gotchas, and patterns worth preserving.

## Current State

The dashboard UI foundation and first real feature (Hetzner Cloud provider) are complete. Users can configure a Hetzner API token at `/settings?tab=providers`, the backend validates it via hcloud-go, and it's persisted in SQLite.

**What's built:**
- Go HTTP server with embedded static assets (`embed.FS`)
- SQLite database layer (`modernc.org/sqlite`, pure Go) with goose embedded migrations
- Provider abstraction: `Provider` interface + global registry + `Store` for DB CRUD
- Hetzner Cloud implementation: token validation via Location.List + Server.List
- 6 API endpoints: health, provider CRUD, standalone validation, provider types
- Nuxt 4 frontend with Tailwind v4 dark theme
- Full OKLCH design system with surface/accent/semantic color scales
- 11 reusable UI components, 3 composables (useModal, useDropdown, useProviders)
- `SettingsProviders` component: empty state, provider list, add modal with 2-step validate→save flow, inline tutorial
- 3 pages: Dashboard (placeholder), Settings (Providers functional, 6 sections placeholder), Components (UI showcase)
- Responsive layout with top nav, mobile hamburger menu, footer
- `make build` produces a single binary, `make dev` runs both servers with hot reload

## Next Steps

| Item | Priority | Notes |
|------|----------|-------|
| Server provisioning | High | Use stored provider credentials to actually create/manage Hetzner servers |
| More providers | Medium | DigitalOcean, AWS, Vultr — implement `Provider` interface + blank import |
| Dashboard page content | High | Real monitoring/overview widgets (server status, resource usage) |
| Settings section content | Medium | Fill remaining 6 sections (General, Servers, Sites, Notifications, Security, API Keys) |
| Token encryption at rest | Low | Currently plaintext in SQLite — acceptable for local app MVP |
| Provider health checks | Low | Periodic re-validation of stored tokens |
| Additional components | Medium | Tables, toasts/notifications as needed |

## Gotchas for Future Sessions

### Tailwind v4 + Nuxt 4

- **DO NOT** install `@nuxtjs/tailwindcss` — it's Tailwind v3 only. Use `@tailwindcss/vite` as a Vite plugin instead.
- Tailwind v4 has **no `tailwind.config.js`**. All config is CSS-first via `@theme {}` blocks in `main.css`.
- Import is `@import "tailwindcss"` (not `@import "tailwindcss/base"` etc.)

### Build Pipeline

- `nuxt generate` (not `nuxt build`) for static output
- Output goes to `web/.output/public/` → copied to `internal/server/dist/` → embedded in Go binary
- The `internal/server/dist/` directory has a `.gitkeep` — actual assets are generated, not committed

### Nuxt 4 Specifics

- App directory is `web/app/` (Nuxt 4 default), not `web/` root
- Pages, components, composables, layouts all live under `web/app/`
- `app.vue` uses `<NuxtLayout><NuxtPage /></NuxtLayout>` pattern

### hcloud-go Version Constraint

- System has Go 1.22. hcloud-go v2.20+ requires Go 1.23+.
- **Pinned to v2.19.0** (requires go 1.21, compatible with 1.22). API surface is identical.
- When upgrading to Go 1.23+, unpin to latest.

### SQLite + Goose

- `modernc.org/sqlite` is pure Go — no CGo, no C compiler needed. Use driver name `"sqlite"` (not `"sqlite3"`).
- Goose embedded migrations: **must** use `fs.Sub(embedMigrations, "migrations")` to strip the directory prefix, otherwise goose reports "no migrations found".
- SQLite pragmas applied on every connection: WAL, foreign_keys, busy_timeout, synchronous=NORMAL.
- `MaxOpenConns(1)` — SQLite only supports one writer at a time.

### Dev Proxy

- Nuxt dev server proxies `/api` to Go backend via `nitro.devProxy` in `nuxt.config.ts`
- Default Go port: 8081, default Nuxt port: 8080
- Configurable via `DEV_API_PORT` and `DEV_UI_PORT` make variables

## Patterns Worth Preserving

### Component Architecture

- All UI components in `components/ui/` with `Ui` prefix (auto-imported by Nuxt)
- Props-driven with TypeScript interfaces
- Slots for composition (header/body/footer in UiCard, trigger in UiDropdown, label in UiProgressBar)
- Composables extract reusable logic (useModal, useDropdown)

### Design System

- OKLCH colors defined in `@theme {}` block in `main.css`
- Surface scale 950→50 for dark theme (950 = background, 50 = brightest text)
- Semantic colors: success (green), warning (amber), danger (red), info (accent)
- Custom utilities: `glass` (frosted backdrop), `glow-accent`, `glow-primary`
- Font stack: Inter (sans), JetBrains Mono (mono) — self-hosted

### Provider Pattern

- Adding a new provider: implement `Provider` interface in `internal/provider/{name}/`, call `provider.Register()` in `init()`, add blank import `_ "pressluft/internal/provider/{name}"` in `cmd/main.go`. Zero changes to existing code.
- Token validation: no dedicated "verify" endpoint for most cloud APIs. Use a lightweight read call (e.g. list locations) for auth check, then a resource-specific call for permission check.
- `StoredProvider.APIToken` has `json:"-"` — never serialized to API responses.

### Page Patterns

- **Placeholder pages**: `<h1>` headline + `<p>` subline, minimal template-only SFC
- **Feature pages**: `<script setup>` with composables + reactive state, sections with `<UiCard>` wrappers
- **Settings sections**: Extract into dedicated components (e.g. `SettingsProviders.vue`) when section has real functionality. Keep placeholder sections inline in `settings.vue`.
- **Sub-navigated pages**: Query-param routing (`?tab=section`) with vertical sidebar on desktop, collapsible dropdown on mobile. Sections defined as a typed array, active section derived from `useRoute().query`. See `settings.vue` as the reference implementation. Prefer this over nested file-based routes for in-page section switching.

## 📂 Codebase References

- `web/app/components/SettingsProviders.vue` - Current implemented providers flow and UI states
- `web/app/composables/useProviders.ts` - Frontend calls for providers API
- `internal/server/handler_providers.go` - Backend provider endpoint behavior and response shape
- `internal/provider/hetzner/hetzner.go` - Hetzner validation behavior and permission messaging
- `internal/database/database.go` - Runtime DB setup and migration execution
- `cmd/main.go` - DB path resolution (`PRESSLUFT_DB`, XDG fallback)

## Related Files

- `technical-domain.md` — Full stack and architecture details
- `decisions-log.md` — Why things were done this way
