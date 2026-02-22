<!-- Context: development/concepts | Priority: high | Version: 1.0 | Updated: 2026-02-22 -->

# Concept: Nuxt Deployment Mode Behind Go

For a Go-hosted dashboard, static Nuxt output (`.output/public`) is usually the lean default, while SSR requires a separate Node runtime. Choose SSR only when route-time rendering is required.

## Key Points

- Static mode: `nuxt generate` output is served directly by Go.
- SSR mode: `nuxt build` needs Node running `.output/server/index.mjs`.
- Route rules (`prerender`, `ssr`) support hybrid behavior when needed.
- Static mode is operationally simpler for internal/admin dashboards.

## Minimal Example

```ts
export default defineNuxtConfig({
  ssr: true,
  nitro: { prerender: { routes: ['/'] } },
})
```

## References

- Source archive: `.tmp/archive/harvested/2026-02-22/external-context/nuxt/production-build-behind-go-server.md`
- Docs: https://nuxt.com/docs/4.x/getting-started/deployment

## Codebase References

- `web/nuxt.config.ts` - Nuxt render/build configuration
- `web/package.json` - Build scripts used in CI/local
- `README.md` - Deployment and run instructions

## Related

- `concepts/go-embedded-dashboard.md`
- `guides/static-dashboard-build-flow.md`
