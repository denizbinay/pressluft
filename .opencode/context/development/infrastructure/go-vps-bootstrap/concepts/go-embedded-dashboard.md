<!-- Context: development/concepts | Priority: critical | Version: 1.0 | Updated: 2026-02-22 -->

# Concept: Go Embedded Dashboard Assets

Go can ship frontend assets inside the binary using `embed.FS`, then serve them with `http.FileServer`. This removes runtime file dependencies and fits VPS deployments that prefer a single executable.

## Key Points

- `//go:embed` patterns must resolve at build time from the package directory.
- Use `fs.Sub` to scope to the static output directory before serving.
- Prefer explicit server config (`http.Server` with timeouts) over bare `ListenAndServe`.
- Keep SPA fallback behavior explicit instead of routing all 404s to `index.html`.

## Minimal Example

```go
//go:embed web/.output/public/*
var raw embed.FS
sub, _ := fs.Sub(raw, "web/.output/public")
mux := http.NewServeMux()
mux.Handle("/", http.FileServer(http.FS(sub)))
```

## References

- Source archive: `.tmp/archive/harvested/2026-02-22/external-context/go/single-binary-embedded-assets.md`
- Docs: https://pkg.go.dev/embed

## Codebase References

- `cmd/main.go` - HTTP server and route wiring
- `web/.output/public/` - Static dashboard artifact location
- `Makefile` - Build flow for generated assets

## Related

- `concepts/nuxt-go-deployment-mode.md`
- `guides/static-dashboard-build-flow.md`
