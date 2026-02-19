Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/state-machines.md, docs/job-execution.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: environment-deploy-updates

## Problem

Operators need deterministic deploy and update workflows with health-gated rollout and rollback safety.

## Scope

- In scope:
  - Implement `POST /api/environments/:id/deploy` and `POST /api/environments/:id/updates`.
  - Enqueue `env_deploy` and `env_update` jobs.
  - Enforce state transitions and health-check integration.
  - Require a pre-update backup snapshot before `env_update` execution.
- Out of scope:
  - Canary/blue-green deployment strategies.
  - Multi-node traffic shifting.

## Allowed Change Paths

- `internal/api/**`
- `internal/environments/**`
- `internal/jobs/**`
- `internal/store/**`
- `ansible/playbooks/env-deploy.yml`
- `ansible/playbooks/env-update.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-environment-deploy-updates.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Deploy endpoint validates `source_type` and `source_ref`, returns `202` with `job_id`.
2. Updates endpoint validates scope and returns `202` with `job_id`.
3. Deploy/update jobs set environment/site to `deploying` and return to `active` or `failed` transactionally.
4. Health check failure triggers rollback path per architecture spec.
5. Update jobs are blocked unless a fresh pre-update backup exists or is created first.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/env-deploy.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/env-update.yml`
- Required tests:
  - Deploy and updates handler tests.
  - Job state transition tests for deploying lifecycle.

## Risks and Rollback

- Risk: failed deploy may leave wrong `current_release_id`.
- Rollback: force release_rollback job and reconcile release pointer in DB transaction.
