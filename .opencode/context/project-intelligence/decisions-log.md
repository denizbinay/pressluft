<!-- Context: project-intelligence/decisions | Priority: high | Version: 1.3 | Updated: 2026-02-23 -->

# Decisions Log

> Major architectural and technical decisions with full context.

---

## Decision: Tailwind CSS v4 via Vite Plugin (not @nuxtjs/tailwindcss)

**Date**: 2026-02-22
**Status**: Decided

### Context

Needed Tailwind CSS for the Nuxt 4 dashboard. The `@nuxtjs/tailwindcss` module exists but is still on Tailwind v3.

### Decision

Use `@tailwindcss/vite` as a Vite plugin configured in `nuxt.config.ts` under `vite.plugins`, instead of the `@nuxtjs/tailwindcss` Nuxt module.

### Rationale

Tailwind v4 uses a fundamentally different configuration model (CSS-first with `@theme {}` blocks instead of `tailwind.config.js`). The official Nuxt module hasn't been updated for v4 yet. The Vite plugin is the official Tailwind v4 integration path and works directly with Nuxt's Vite pipeline.

### Alternatives Considered

| Alternative | Pros | Cons | Why Rejected? |
|-------------|------|------|---------------|
| `@nuxtjs/tailwindcss` module | Nuxt-native, auto-config | Still Tailwind v3, no v4 support | Outdated version |
| PostCSS plugin | Works everywhere | More manual setup | Vite plugin is simpler for Vite-based projects |

### Impact

- **Positive**: Access to Tailwind v4 features (OKLCH, CSS-first config, `@theme {}`)
- **Negative**: Can't use `@nuxtjs/tailwindcss` module conveniences (auto-imports, viewer)
- **Risk**: When `@nuxtjs/tailwindcss` updates to v4, may want to migrate back

---

## Earlier Decisions (2026-02-22 — 2026-02-23)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| OKLCH colors | OKLCH for all design system colors in `@theme {}` | Perceptually uniform lightness, native Tailwind v4 support |
| Self-hosted fonts | `@nuxtjs/google-fonts` with `download: true` | Works offline (single-binary may run without internet), no CDN dependency |
| Static generation | `nuxt generate` (not SSR) | No Node runtime in production, single binary, admin UI doesn't need SSR |
| Three-page nav | Dashboard, Settings, Components | Don't create feature pages until features exist; add pages when real features are built |

---

## Decision: Query-Param Sub-Navigation (not Nested Routes)

**Date**: 2026-02-23
**Status**: Decided

### Context

The Settings page needs multiple sections (General, Providers, Servers, Sites, Notifications, Security, API Keys). Two approaches: nested file-based routes (`pages/settings/general.vue`, `pages/settings/providers.vue`, etc.) or a single page with query-param switching (`/settings?tab=general`).

### Decision

Use a single `settings.vue` page with query-param routing (`?tab=general`). Sections defined as a typed array. Active section derived from `useRoute().query` with a computed property. Desktop shows a vertical sidebar; mobile collapses to a dropdown selector.

### Rationale

All settings sections share the same layout (sidebar + content card). Nested routes would duplicate this layout or require a settings-specific layout file. Query params keep it in one file, make the sidebar state trivial (just a computed from the route), and avoid Nuxt's nested route complexity for what is essentially tab switching. The URL is still bookmarkable and shareable.

### Alternatives Considered

| Alternative | Pros | Cons | Why Rejected? |
|-------------|------|------|---------------|
| Nested file routes (`pages/settings/*.vue`) | Nuxt-native, code-split per section | Layout duplication, more files, overkill for placeholder sections | Unnecessary complexity at this stage |
| Component-only tabs (no URL) | Simplest | Not bookmarkable, no URL state | Bad UX — can't link to a specific section |

### Impact

- **Positive**: Single file, bookmarkable URLs, trivial to add new sections (just add to the array), shared sidebar/layout
- **Negative**: All sections in one file (could get large when content is real)
- **Risk**: If sections grow very large, can extract each section's content into its own component and lazy-import. The pattern still holds.

---

## Decision: SQLite via modernc.org/sqlite (Pure Go, No CGo)

**Date**: 2026-02-23
**Status**: Decided

### Context

Needed local persistence for provider credentials. Options: SQLite (via CGo or pure Go), BoltDB, or flat files.

### Decision

Use `modernc.org/sqlite` — a pure Go SQLite implementation with no CGo dependency.

### Rationale

Single-binary deployment is a core constraint. CGo-based SQLite (`mattn/go-sqlite3`) requires a C compiler and complicates cross-compilation. `modernc.org/sqlite` is a transpiled pure Go implementation that works with `database/sql` and requires no external toolchain. Performance is sufficient for a local app with low write volume.

### Alternatives Considered

| Alternative | Pros | Cons | Why Rejected? |
|-------------|------|------|---------------|
| `mattn/go-sqlite3` (CGo) | Battle-tested, faster | Requires C compiler, breaks cross-compile | Conflicts with single-binary goal |
| BoltDB/bbolt | Pure Go, embedded | Key-value only, no SQL, no migrations | Too low-level for relational data |
| Flat files (JSON/YAML) | Simplest | No queries, no transactions, no schema | Doesn't scale past trivial use |

### Impact

- **Positive**: Zero external dependencies, cross-compiles cleanly, `database/sql` compatible
- **Negative**: Slightly slower than CGo SQLite (acceptable for local app)
- **Config**: `MaxOpenConns(1)`, WAL mode, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=NORMAL`

---

## Decision: Goose v3 for Embedded SQL Migrations

**Date**: 2026-02-23
**Status**: Decided

### Context

Need schema migrations for the SQLite database. Migrations should be embedded in the binary (no external files at runtime).

### Decision

Use `pressly/goose/v3` with `//go:embed migrations/*.sql` and `goose.NewProvider()`.

### Rationale

Goose supports embedded filesystems natively via `NewProvider()`. SQL-based migrations are readable and auditable. The provider API runs migrations on startup automatically.

### Impact

- **Positive**: Migrations embedded in binary, runs on startup, SQL-based (readable)
- **Gotcha**: Must use `fs.Sub(embedMigrations, "migrations")` to strip the directory prefix — otherwise goose reports "no migrations found"

---

## Decision: hcloud-go v2.19.0 (Pinned for Go 1.22 Compatibility)

**Date**: 2026-02-23
**Status**: Decided

### Context

Need the official Hetzner Cloud Go SDK. Latest versions (v2.20+) require Go 1.23+, but the project uses Go 1.22.

### Decision

Pin to `hcloud-go v2.19.0` (requires go 1.21, compatible with 1.22). The API surface is identical to newer versions for our use case.

### Impact

- **Positive**: Works with Go 1.22, full API coverage for validation and server management
- **Risk**: When upgrading to Go 1.23+, can unpin to latest. No breaking changes expected.

---

## Decision: Provider Abstraction with Interface + Registry + init() Auto-Registration

**Date**: 2026-02-23
**Status**: Decided

### Context

Need an extensible architecture for cloud providers. Only Hetzner for MVP, but must be obvious how to add more.

### Decision

`Provider` interface (`Info()` + `Validate()`) with a global registry (`Register()`/`Get()`/`All()`). Providers self-register via `init()` and are activated by blank import in `cmd/main.go` (`_ "pressluft/internal/provider/hetzner"`).

### Rationale

Adding a new provider = implement the interface + add a blank import. No factory switches, no config files. The `init()` pattern is idiomatic Go for plugin-style registration (used by `database/sql` drivers, image codecs, etc.).

### Impact

- **Positive**: Adding a provider is 1 file + 1 import line. Zero changes to existing code.
- **Negative**: Global mutable state (registry map) — acceptable for a single-binary app.

---

## 📂 Codebase References

- `web/nuxt.config.ts` - Tailwind v4 Vite plugin, Google Fonts, API dev proxy
- `web/app/assets/css/main.css` - OKLCH color tokens and theme utilities
- `web/app/pages/settings.vue` - Query-param section routing pattern
- `internal/database/database.go` - SQLite setup and embedded goose provider
- `internal/database/migrations/00001_create_providers.sql` - Migration source for providers table
- `internal/provider/provider.go` - Registry design and provider contract
- `internal/provider/hetzner/hetzner.go` - hcloud-go usage and validation checks

## Related Files

- `technical-domain.md` — Technical implementation details
- `living-notes.md` — Current state and next steps
