<!-- Context: project-intelligence/notes | Priority: high | Version: 1.1 | Updated: 2026-02-23 -->

# Living Notes

> Active state, next steps, gotchas, and patterns worth preserving.

## Current State

The dashboard UI foundation is complete. All infrastructure (Go backend, Nuxt frontend, build pipeline, design system) is working. No real features implemented yet — pages are placeholders ready for content.

**What's built:**
- Go HTTP server with embedded static assets (`embed.FS`)
- Nuxt 4 frontend with Tailwind v4 dark theme
- Full OKLCH design system with surface/accent/semantic color scales
- 11 reusable UI components, 2 composables
- 3 pages: Dashboard (placeholder), Settings (placeholder), Components (UI library showcase)
- Responsive layout with top nav, mobile hamburger menu, footer
- `make build` produces a single binary, `make dev` runs both servers with hot reload

## Next Steps

| Item | Priority | Notes |
|------|----------|-------|
| Dashboard page content | High | Real monitoring/overview widgets |
| Settings page content | Medium | Configuration UI |
| API integration | High | Connect frontend to Go `/api` routes |
| Additional components | Medium | Tables, tabs, toasts/notifications, sidebar panels as needed |
| Health check widget | Low | Was in old `app.vue`, could be re-integrated into layout or dashboard |

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

### Page Pattern

- Placeholder pages: `<h1>` headline + `<p>` subline, minimal template-only SFC
- Feature pages: `<script setup>` with composables + reactive state, sections with `<UiCard>` wrappers

## Related Files

- `technical-domain.md` — Full stack and architecture details
- `decisions-log.md` — Why things were done this way
