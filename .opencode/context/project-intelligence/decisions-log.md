<!-- Context: project-intelligence/decisions | Priority: high | Version: 1.2 | Updated: 2026-02-23 -->

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

## Decision: OKLCH Color Format for Design System

**Date**: 2026-02-22
**Status**: Decided

### Context

Needed a color system for the dark-themed dashboard. Traditional hex/HSL colors have perceptual uniformity issues.

### Decision

Use OKLCH color format for all design system colors defined in `main.css` `@theme {}` blocks.

### Rationale

OKLCH provides perceptually uniform lightness, making it easier to create consistent color scales. Tailwind v4 has native OKLCH support. Modern browsers support it well.

### Impact

- **Positive**: Perceptually uniform color scales, consistent contrast ratios
- **Negative**: Less familiar to developers used to hex/HSL
- **Risk**: Minimal — fallback is automatic in modern browsers

---

## Decision: Self-Hosted Google Fonts

**Date**: 2026-02-22
**Status**: Decided

### Context

Using Inter (UI) and JetBrains Mono (code) fonts. Could load from Google CDN or self-host.

### Decision

Self-host via `@nuxtjs/google-fonts` module with `download: true` and `inject: true`.

### Rationale

Single-binary deployment means the dashboard may run on internal networks without internet access. Self-hosting eliminates external CDN dependency and improves privacy (no Google tracking). Also better for performance (no DNS lookup, no CORS).

### Impact

- **Positive**: Works offline, no external dependencies, better privacy
- **Negative**: Slightly larger build output
- **Risk**: None significant

---

## Decision: Static Generation (not SSR) for Go Embedding

**Date**: 2026-02-22
**Status**: Decided

### Context

Nuxt supports SSR (requires Node runtime) and static generation (pure HTML/JS/CSS). The dashboard is embedded in a Go binary.

### Decision

Use `nuxt generate` for static output. Go serves the files via `embed.FS`.

### Rationale

No Node runtime needed in production. Single binary deployment. The dashboard is an admin/monitoring UI — no SEO or dynamic server rendering needed.

### Alternatives Considered

| Alternative | Pros | Cons | Why Rejected? |
|-------------|------|------|---------------|
| SSR mode | Dynamic rendering, better SEO | Requires Node runtime alongside Go | Unnecessary complexity for admin dashboard |
| SPA mode | Simpler | No prerendering benefits | Static gen gives both |

### Impact

- **Positive**: Single binary, no Node in production, simpler deployment
- **Negative**: No server-side rendering (acceptable for admin UI)

---

## Decision: Three-Page Navigation Structure

**Date**: 2026-02-23
**Status**: Decided

### Context

Initially had 4 pages (Dashboard, Pipelines, Services, Settings). The kitchen-sink component showcase was on the Dashboard page. Pipelines and Services were premature placeholders.

### Decision

Restructured to 3 pages: Dashboard (empty placeholder), Settings (empty placeholder), Components (UI library showcase). Removed Pipelines and Services pages.

### Rationale

Don't create feature pages until features exist. The component showcase is useful as a living reference but shouldn't be the dashboard. Keep navigation clean and add pages when real features are implemented.

### Impact

- **Positive**: Cleaner navigation, component library preserved as reference
- **Negative**: None
- **Risk**: None — pages can be added back when features are built

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

## Related Files

- `technical-domain.md` — Technical implementation details
- `living-notes.md` — Current state and next steps
