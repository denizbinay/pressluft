Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/job-execution.md, docs/api-contract.md, docs/pre-plan-readiness.md
Supersedes: none

# Testing

This document defines required verification gates for changes.

## Global Gates

- Before commit: `go build -o ./bin/pressluft ./cmd/pressluft` and `go vet ./...` must pass.
- Before PR: `go test ./internal/... -v` must pass.

## Frontend Gates

Run from repository root:

- `cd web && pnpm install`
- `cd web && pnpm lint`
- `cd web && pnpm build`

Dashboard embed expectation:

- `pressluft serve` reads built assets from `./web/.output/public` by default.
- Embedded dashboard serving requires `./web/.output/public/index.html` to exist.
- Override asset location with `PRESSLUFT_WEB_DIST_DIR` or `-web-dist-dir` when needed.
- Build frontend assets before smoke-testing embedded UI routes.

Embedded dashboard smoke check (release readiness):

- `cd web && pnpm build`
- `go build -o ./bin/pressluft ./cmd/pressluft`
- Prepare an empty DB with schema:
  - `PRESSLUFT_DB_PATH=$(mktemp) ./bin/pressluft migrate up`
- Start server (example):
  - `PRESSLUFT_DB_PATH=$PRESSLUFT_DB_PATH PRESSLUFT_SECRETS_DIR=$(mktemp -d) ./bin/pressluft -listen :18080 serve`
- Verify HTML routes return 200:
  - `curl -fsS http://127.0.0.1:18080/ >/dev/null`
  - `curl -fsS http://127.0.0.1:18080/login >/dev/null`
  - `curl -fsS http://127.0.0.1:18080/app >/dev/null`

## Change-Type Requirements

### Backend behavior change

- Add or update at least one unit or integration test near affected package.
- Test must assert behavior defined in governing feature spec.

### API contract change

- Update `contracts/openapi.yaml`.
- Update relevant tests for request validation and response shape.

### DB schema or migration change

- Add migration tests or migration verification steps.
- Validate forward migration path (`go run ./migrations/migrate.go up`).

### Ansible/playbook change

- Run `ansible-playbook --syntax-check` for affected playbooks.
- Run `ansible-lint` when available.

## Acceptance Mapping

Every substantial PR should map acceptance criteria to verification evidence:

- criterion -> command/test -> pass/fail summary

## Spec and Contract Readiness Checks

Before implementation planning starts, run lightweight checks:

- Metadata presence check across `docs/*.md`.
- Endpoint ownership parity check between `contracts/openapi.yaml` and `docs/contract-traceability.md`.
- Error code registry check between contract/docs and `docs/error-codes.md`.
- Job type registry check between traceability/ansible and `docs/job-types.md`.
- Job type ownership check: each canonical job type maps to one feature spec.
- Enum/state alignment check across:
  - `contracts/openapi.yaml`
  - `docs/data-model.md`
  - `docs/state-machines.md`

## Scripted Checks

The following scripts automate readiness and drift checks:

- `bash scripts/check-readiness.sh`
- `bash scripts/check-contract-traceability.sh`
- `bash scripts/check-job-error-registry.sh`
- `bash scripts/check-parallel-locks.sh`

`scripts/check-readiness.sh` also validates OpenCode runtime bootstrap (`opencode.json`) and required local agent role files under `.opencode/agents/`.

## OpenCode Command Presets

Project command presets under `.opencode/commands/` provide deterministic gate execution in-session:

- `/readiness`
- `/backend-gates`
- `/frontend-gates`
- `/session-kickoff <docs/features/feature-*.md>`
- `/run-plan`
- `/resume-run`
- `/triage-failures`

Smoke-test these commands after OpenCode config changes to confirm command discovery and output wiring remain functional.

## Local Sandbox

- Disposable local run (auto temp DB + secrets): `bash scripts/dev-sandbox.sh`

## CI Expectations

- CI must run scripted checks on each PR/push.
- CI must run backend gates when Go bootstrap files exist.
- CI status is blocking for merge once backend gates become runnable.
