<!-- Context: project-intelligence/technical | Priority: high | Version: 1.1 | Updated: 2026-02-23 -->

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
├── cmd/main.go                    # Go entrypoint, HTTP server
├── internal/server/
│   ├── handler.go                 # Route handlers
│   ├── handler_test.go
│   ├── logging.go                 # Request logging middleware
│   ├── logging_test.go
│   └── dist/                      # Embedded static assets (generated, gitkeep)
├── web/                           # Nuxt 4 frontend
│   ├── nuxt.config.ts             # Tailwind v4 vite plugin, Google Fonts, API proxy
│   ├── package.json
│   ├── app/
│   │   ├── app.vue                # Root: <NuxtLayout><NuxtPage /></NuxtLayout>
│   │   ├── assets/css/main.css    # Design system: OKLCH theme, custom utilities
│   │   ├── layouts/default.vue    # Top nav, content area, footer, mobile menu
│   │   ├── composables/           # useModal, useDropdown
│   │   ├── components/ui/         # 11 reusable UI components (UiButton, UiCard, etc.)
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
| `/settings` | Settings | Placeholder (headline + subline) |
| `/components` | UI Components | Kitchen-sink showcase of all UI components |

### UI Components (11)

UiButton (5 variants, 3 sizes, loading/disabled), UiCard (slots, hoverable), UiBadge (5 variants), UiProgressBar (4 colors, 3 sizes), UiInput, UiSelect, UiTextarea, UiToggle, UiModal (teleported, animated), UiDropdown (click-outside, escape), UiDropdownItem (normal/danger/disabled)

### Composables (2)

`useModal()` — open/close/toggle reactive state
`useDropdown()` — click-outside and escape key handling

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

## Related Files

- `business-domain.md` — Why this project exists
- `decisions-log.md` — Key technical decisions with rationale
- `living-notes.md` — Current state, next steps, gotchas
