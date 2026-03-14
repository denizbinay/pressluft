Read `README.md` first.

The project uses a unified CLI (`pressluft`) for all development and build tasks.
Bootstrap it with `make`, then use `pressluft <command>` for everything.

The CLI uses Cobra — run `pressluft help` or `pressluft <command> --help`
to discover commands, flags, and usage. The `cobra.Command` structs in
`cmd/pressluft/` are the source of truth for the command surface.

Do not hand-edit generated files:
- `web/app/lib/api-contract.ts`
- `web/app/lib/platform-contract.generated.ts`

These are regenerated automatically by `pressluft dev` and `pressluft build`.

Ignore generated/local directories during search unless the task explicitly needs them:
- `web/.output`
- `web/.nuxt`
- `.venv`

The three binaries are:
- `cmd/pressluft/` — the CLI (dev tools, build, doctor)
- `cmd/pressluft-server/` — the control-plane server (runtime)
- `cmd/pressluft-agent/` — the server agent (runtime)

CLI output is styled via `internal/cliui/` (Lip Gloss v2).
Colors degrade automatically when stdout is not a TTY.

Local dev state lives in `.pressluft/` at the repo root (SQLite DB, keys).
To reset: `rm -rf .pressluft`
