Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/job-execution.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: jobs-and-metrics

## Problem

Operators need visibility into background work and platform health to trust automation.

## Scope

- In scope:
  - Implement `GET /api/jobs`, `GET /api/jobs/{id}`, and `GET /api/metrics`.
  - Return canonical job status and error fields.
  - Return point-in-time metrics counters defined by contract.
- Out of scope:
  - Prometheus scrape endpoint.
  - Historical analytics/time-series dashboards.

## Allowed Change Paths

- `internal/api/**`
- `internal/jobs/**`
- `internal/metrics/**`
- `internal/store/**`
- `internal/devserver/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-jobs-and-metrics.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Jobs list endpoint returns queued/running/completed jobs with stable payload shape.
2. Job detail endpoint returns full job state including attempts and error fields.
3. Metrics endpoint returns non-negative counters for running/queued jobs, active nodes, and total sites.
4. Unauthorized requests to all three endpoints return `401`.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Jobs list/detail handler tests.
  - Metrics aggregation tests.

## Risks and Rollback

- Risk: stale metrics semantics can mislead operators.
- Rollback: lock metrics to direct DB counters only until richer model is designed.
