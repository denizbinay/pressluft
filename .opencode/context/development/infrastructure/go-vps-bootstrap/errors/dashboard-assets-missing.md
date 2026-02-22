<!-- Context: development/errors | Priority: medium | Version: 1.0 | Updated: 2026-02-22 -->

# Error: Dashboard Assets Missing

This appears when the Go server starts but embedded dashboard files are absent or stale. The most common cause is running the binary without generating/copying Nuxt static output first.

## Key Points

- Symptom: root page falls back to "Dashboard assets not found".
- Cause: `internal/server/dist/index.html` missing at build time.
- Fix: regenerate Nuxt static files and rebuild Go binary.
- Prevention: run `make check` before release commits.
- Guardrail: root-level `pnpm` fails in this repo; run package scripts under `web/`.

## Minimal Example

```bash
make build
./bin/pressluft
curl -i http://127.0.0.1:8080/
```

## References

- `../../../../../../README.md`
- `../../../../../../docs/bootstrap-validation.md`

## Related

- `../guides/static-dashboard-build-flow.md`
- `../concepts/go-embedded-dashboard.md`
