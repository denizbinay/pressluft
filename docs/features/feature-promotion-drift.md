Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/spec-index.md, docs/promotion-and-drift.md, docs/state-machines.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: promotion-drift

## Problem

Operators must promote changes safely while protecting live production data from overwrite.

## Scope

- In scope:
  - Implement `POST /api/environments/{id}/drift-check` and `POST /api/environments/{id}/promote`.
  - Enforce drift check and fresh backup gate before promotion.
  - Enqueue `drift_check` and `env_promote` jobs.
- Out of scope:
  - Arbitrary custom diff engines.
  - Multi-target fanout promotions.

## Allowed Change Paths

- `internal/api/**`
- `internal/promotion/**`
- `internal/jobs/**`
- `internal/store/**`
- `ansible/playbooks/drift-check.yml`
- `ansible/playbooks/env-promote.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-promotion-drift.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Drift-check endpoint returns `202` with `job_id` and persists drift record.
2. Promote endpoint blocks when drift gate or backup gate requirements are unmet.
3. Promote endpoint exposes no admin override path when gates are unmet.
4. Successful promotion respects preset-based protected resources.
5. Promotion job failures leave clear non-success state and error fields.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/drift-check.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/env-promote.yml`
- Required tests:
  - Drift-check and promote handler tests.
  - Gate enforcement tests (drift and backup).

## Risks and Rollback

- Risk: incorrect drift classification allows unsafe overwrite.
- Rollback: disable promote path with stable API error until drift logic is corrected.
