<!-- Context: development/examples | Priority: medium | Version: 1.0 | Updated: 2026-02-22 -->

# Example: Minimal Embed Handler

Use this baseline when you need a single Go binary to serve static dashboard files. It demonstrates the smallest safe wiring for `embed.FS` + `http.FileServer`.

## Key Points

- Keep embed path stable to avoid runtime missing-file surprises.
- Scope embedded FS with `fs.Sub` before mounting.
- Serve root from embedded static directory.
- Pair this with explicit health routes in a separate mux branch.

## Minimal Example

```go
//go:embed dist/*
var assets embed.FS
sub, _ := fs.Sub(assets, "dist")
mux := http.NewServeMux()
mux.Handle("/", http.FileServer(http.FS(sub)))
```

## References

- https://pkg.go.dev/embed
- https://pkg.go.dev/io/fs#Sub

## Related

- `../concepts/go-embedded-dashboard.md`
- `../guides/static-dashboard-build-flow.md`
