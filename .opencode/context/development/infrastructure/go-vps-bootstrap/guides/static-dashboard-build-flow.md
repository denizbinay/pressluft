<!-- Context: development/guides | Priority: high | Version: 1.0 | Updated: 2026-02-22 -->

# Guide: Static Dashboard Build Flow

Build the Nuxt dashboard as static output and serve it from Go so deployment stays single-runtime. Keep the workflow repeatable for local and VPS builds.

## Key Points

- Install frontend dependencies before generating static output.
- Run static generation and publish only `.output/public` for Go serving.
- Keep output path stable so Go embed paths do not drift.
- Document commands in repo-level bootstrap docs.
- Run repo validation from root with `make check` (or `make lint`, `make test`, `make build`).
- Do not run root-level `pnpm` for app scripts; package scripts live in `web/package.json`.

## Minimal Example

```bash
cd web
npm ci
npm run generate
cd ..
go build ./cmd
```

Validation commands:

```bash
make check
# or stepwise
make lint && make test && make build
```

If you need package-manager commands directly, scope them to `web/`:

```bash
pnpm --dir web lint
pnpm --dir web build
```

## References

- Source archive: `.tmp/archive/harvested/2026-02-22/external-context/nuxt/dashboard-scaffold-static-generation-go-hosting.md`
- Docs: https://nuxt.com/docs/4.x/getting-started/prerendering

## Codebase References

- `Makefile` - Repeatable build and check commands
- `web/package.json` - Nuxt install/build scripts
- `docs/bootstrap-validation.md` - Validation workflow

## Related

- `concepts/nuxt-go-deployment-mode.md`
- `lookup/ansible-execution-guardrails.md`
