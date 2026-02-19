Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/job-execution.md, docs/state-machines.md, docs/api-contract.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: job-control

## Problem

Operators need explicit control over stuck or failed operations through cancel and reset actions with safe state transitions.

## Scope

- In scope:
  - Define and implement admin job cancellation behavior.
  - Define and implement explicit reset action for site/environment `failed -> active` transitions after validation.
  - Ensure job and resource state transitions are transactionally safe.
- Out of scope:
  - Automatic policy-based cancellation.
  - Bulk reset/cancel operations.

## Allowed Change Paths

- `internal/api/**`
- `internal/jobs/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/store/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/state-machines.md`
- `docs/features/feature-job-control.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/state-machines.md`

## Acceptance Criteria

1. Admin can cancel a queued or running job with deterministic status/result semantics.
2. Cancellation enforces safe-stop behavior for running operations where supported.
3. Explicit reset action is required for `failed -> active` transitions on site/environment resources.
4. Unauthorized or invalid reset/cancel requests return stable structured errors.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Job cancel handler tests for queued and running jobs.
  - Failed-state reset validation tests for sites/environments.

## Risks and Rollback

- Risk: unsafe cancellation can leave partially mutated infrastructure.
- Rollback: restrict cancellation to queued jobs only until safe-stop coverage is verified.
