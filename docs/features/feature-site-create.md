Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/data-model.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: site-create

## Problem

Operators need one-click site creation that yields an isolated, reachable initial environment.

## Scope

- In scope:
  - Implement `POST /api/sites`, `GET /api/sites`, and `GET /api/sites/{id}`.
  - Enqueue `site_create` jobs for create operations.
  - Create initial production environment with deterministic preview URL.
- Out of scope:
  - Multi-tenant permissions.
  - Custom provisioning workflows outside defined job types.

## Allowed Change Paths

- `internal/api/**`
- `internal/jobs/**`
- `internal/sites/**`
- `internal/store/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/ui-flows.md`
- `docs/features/feature-site-create.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. `POST /api/sites` returns `202` with `job_id` and enqueues `site_create`.
2. Site and initial environment records are persisted with valid status transitions.
3. `GET /api/sites` and `GET /api/sites/{id}` return consistent JSON shapes per contract.
4. Concurrency guard blocks conflicting mutation jobs for the same site.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Create site handler and validation tests.
  - Job enqueue and state transition tests.

## Risks and Rollback

- Risk: partial writes can orphan environments.
- Rollback: revert create flow and run repair migration/tooling for orphaned records.
