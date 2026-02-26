# Pressluft

Single-binary Go service that serves an embedded Nuxt dashboard and exposes bootstrap-safe API endpoints.

## Project Structure

- `cmd/main.go`: Go entrypoint
- `internal/`: internal Go packages (`server`)
- `web/`: Nuxt application source
- `ops/`: operations workspace (profiles, ansible, schemas, scripts)
- `internal/server/dist/`: generated static assets embedded into Go
- `bin/`: compiled binary output

## Prerequisites

- Go 1.22+
- Node.js 20+
- npm 10+

## Build

```bash
make build
```

What this does:
- Installs dashboard dependencies (if missing)
- Generates static Nuxt files to `web/.output/public`
- Copies static files to `internal/server/dist` for embedding
- Builds `bin/pressluft`

## Run

```bash
./bin/pressluft
```

Or run build + start in one step:

```bash
make run
```

Optional port override:

```bash
PORT=9090 ./bin/pressluft
```

## Age Key Management

Pressluft encrypts stored SSH private keys using age. By default it uses
`~/.pressluft/age.key` for the age identity file.

- If `PRESSLUFT_AGE_KEY_PATH` is not set and the default file is missing,
  Pressluft generates a new age identity on first run with permissions `0600`.
- If `PRESSLUFT_AGE_KEY_PATH` is set, the file must already exist and be
  readable; Pressluft will fail fast if it is missing.

Keep the age identity file local to the host running Pressluft and never log
or share the private key contents.

## Dev

```bash
make dev
```

This starts:
- Go backend on `http://localhost:8081`
- Nuxt dev UI on `http://localhost:8080`

`/api` calls from Nuxt are proxied to the Go backend in dev mode.

Go request logs are emitted in structured form (method, path, status, duration).

Optional overrides:

```bash
make dev DEV_API_PORT=8082 DEV_UI_PORT=3000
```

## Test

```bash
make test
```

Runs Go unit tests (`go test ./...`).

## Check

```bash
make check
```

Runs first-commit readiness checks in order:
- format (`go fmt ./...`)
- lint (`go vet ./...`)
- test (`go test ./...`)
- build (`make build`)

## Validation

1. Build succeeds:

```bash
make build
```

2. Root dashboard is served:

```bash
curl -i http://127.0.0.1:8080/
```

Expected: `HTTP/1.1 200 OK` and HTML response body.

3. Health endpoint responds healthy JSON:

```bash
curl -i http://127.0.0.1:8080/api/health
```

Expected body:

```json
{"status":"healthy"}
```

See `docs/bootstrap-validation.md` for a concise validation checklist and bootstrap safety constraints.

## Ops Foundation

- Profile contracts live under `ops/profiles/`
- Ansible convergence scaffolding lives under `ops/ansible/`
- Profile schema lives at `ops/schemas/profile.schema.json`
- Orchestration lifecycle scaffolding is available via `/api/jobs` and `/api/jobs/{id}/events`
