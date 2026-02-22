Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/testing.md, docs/ui-flows.md, docs/features/feature-wp-first-runtime.md, docs/features/feature-wave5-runtime-readiness.md, docs/features/feature-hetzner-sdk-integration.md
Supersedes: none

# FEATURE: wave5-runtime-e2e

## Problem

Wave 5 lacks deterministic end-to-end proof that provider-backed operator flows produce reachable runtimes for clone environment create under real readiness constraints.

## Scope

- In scope:
  - Define deterministic e2e smoke contract for environment clone preview reachability.
  - Define deterministic prerequisite contract that acquires a provider-backed node via `POST /api/nodes` before success-path smokes run.
  - Add/extend scripts to execute full operator flow and assert terminal job states plus preview URL reachability.
  - Add regression coverage for failure classes (preflight blocked, runtime provision failure, unreachable preview).
  - Keep Wave 5 smoke scripts mandatory as closeout gates after SDK migration.
- Out of scope:
  - Browser automation framework migration.
  - Wave 6 deployment checks.

## Allowed Change Paths

- `scripts/**`
- `internal/devserver/**`
- `internal/api/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/jobs/**`
- `docs/testing.md`

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- `docs/ui-flows.md`
- `docs/features/feature-wave5-runtime-e2e.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `docs/ui-flows.md`
- `docs/testing.md`

## Acceptance Criteria

1. Smoke flow acquires or confirms a provider-acquired node target and validates create-environment (clone/staging) success plus preview URL reachability (`200|301|302`).
2. Smoke flow validates backup create/list/restore success paths against the same provider-acquired node target.
3. Smoke scripts fail with deterministic diagnostics including failed command, job id, and reason/error output.
4. Scripts are runnable from repo root and suitable for `/resume-run` verification loops.
5. Provider acquisition pre-provision states are tolerated by polling and succeed without manual retry clicks when eventual readiness is reached.

## Scenarios (WHEN/THEN)

1. WHEN provider acquisition succeeds and node readiness is true THEN clone create and backup/restore smokes pass end-to-end.
2. WHEN readiness preflight blocks creation THEN smoke exits with expected preflight failure semantics.
3. WHEN runtime provisioning fails after enqueue THEN smoke captures terminal failed status and logs actionable context.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `bash scripts/smoke-site-clone-preview.sh`
- Required tests:
  - Script-level assertions for success and expected failure classes.
  - API tests covering smoke-dependent response shapes.

## Risks and Rollback

- Risk: smoke scripts can become flaky due to provider capacity/network timing variance.
- Rollback: keep scripted flow but split strict reachability checks into retried deterministic phases with explicit timeout budgets.
