Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/technical-architecture.md, PLAN.md
Supersedes: none

# FEATURE: wave1-runtime-shell

## Problem

Wave 1 requires a runnable application shell with a visible dashboard placeholder and deterministic logs, but no feature spec currently owns those paths.

## Scope

- In scope:
  - Add `pressluft dev` and `pressluft serve` command entrypoint.
  - Run an HTTP server that serves a Wave 1 placeholder response at `/`.
  - Emit deterministic startup and request logs.
  - Document local developer setup from zero to running.
- Out of scope:
  - Authentication and protected routes.
  - Jobs, metrics, state machines, or infrastructure mutations.

## Allowed Change Paths

- `go.mod`
- `cmd/pressluft/**`
- `internal/devserver/**`
- `README.md`
- `PLAN.md`
- `PROGRESS.md`
- `docs/features/feature-wave1-runtime-shell.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. `go run ./cmd/pressluft dev` starts an HTTP server without additional setup.
2. `GET /` returns the text `Wave 1 complete - features will be added incrementally`.
3. Server logs include deterministic startup and request records with stable field names.
4. Local setup instructions in `README.md` are sufficient to build and run Wave 1 from repository root.

## Scenarios (WHEN/THEN)

1. WHEN an operator runs `pressluft dev` THEN the server starts and listens on the configured port.
2. WHEN a browser requests `/` THEN the placeholder text is returned with HTTP 200.
3. WHEN requests are handled THEN startup and request logs are emitted with deterministic key/value fields.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `go run ./cmd/pressluft dev --port 18080` and `curl http://localhost:18080/`
- Required tests:
  - `internal/devserver/server_test.go`

## Risks and Rollback

- Risk: log format drift can make Wave 1 observability checks brittle.
- Rollback: revert `cmd/pressluft/**`, `internal/devserver/**`, and `go.mod`; restore previous `README.md` instructions.
