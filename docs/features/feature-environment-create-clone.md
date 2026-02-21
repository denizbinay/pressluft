Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/data-model.md, docs/state-machines.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: environment-create-clone

## Problem

Operators need fast, isolated staging/clone environments with clear provenance and state safety.

## Scope

- In scope:
  - Implement `POST /api/sites/{id}/environments` and `GET /api/sites/{id}/environments`.
  - Implement `GET /api/environments/{id}`.
  - Enqueue `env_create` for non-production environment creation.
- Out of scope:
  - Cross-site clone operations.
  - Clone expiration metadata and auto-expiry cleanup.

## Allowed Change Paths

- `internal/api/**`
- `internal/environments/**`
- `internal/jobs/**`
- `internal/store/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/ui-flows.md`
- `docs/features/feature-environment-create-clone.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Environment create returns `202` with `job_id` and persists create intent.
2. Only valid `type` and `promotion_preset` values are accepted.
3. Environment state transitions follow state-machine rules for cloning flow.
4. List and get endpoints return consistent environment representations.
5. Clone create does not introduce expiration metadata in MVP.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Environment create/list/get handler tests.
  - State transition and concurrency lock tests.

## Risks and Rollback

- Risk: invalid source environment handling can create inconsistent clones.
- Rollback: cancel failing job and clean orphaned clone records.
