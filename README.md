Pressluft is an experimental WordPress infrastructure automation platform.
It focuses on reproducible stacks, staging, cloning, optional Git-based workflows,
and Hetzner-first installation via a single script.

Maintainability, security, and boring reliability come before raw speed.

## Local Developer Setup

Prerequisites:

- Go 1.22+

Run from repository root:

1. Start local development server (recommended):

   `make dev`

   Optional port override:

   `make dev PORT=8080`

2. Build and test using repo-local commands:

   - `make build`
   - `make vet`
   - `make test`

3. Open `http://localhost:18080/` and verify the dashboard loads.

Raw Go command equivalents remain available:

- `go run ./cmd/pressluft dev --port 18080`
- `go build -o ./bin/pressluft ./cmd/pressluft`
- `go vet ./...`
- `go test ./internal/... -v`

4. Validate backend gates:

   - `make backend-gates`

## OpenCode Quick Start

This repository is OpenCode-first.

1. Start OpenCode from repository root.
2. Confirm instruction bootstrap is loaded from `opencode.json` and `AGENTS.md`.
3. Run `/readiness` and resolve any failures before implementation.
4. Run `/session-kickoff docs/features/feature-<name>.md` for non-trivial work.
5. Execute implementation with `Spec -> Plan -> Act -> Verify` and keep changes path-scoped.

Useful command presets:

- `/readiness`
- `/session-kickoff docs/features/feature-install-bootstrap.md`
- `/backend-gates`
- `/frontend-gates`

## OpenCode Unattended Quick Start

1. Start OpenCode from repository root (or open this repo in Desktop).
2. Start unattended execution: `/run-plan`.
3. Resume in a new session: `/resume-run`.
4. Triage failures: `/triage-failures`.
