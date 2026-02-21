Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/backups-and-restore.md, docs/state-machines.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: environment-restore

## Problem

Operators need deterministic restore from known backups without cross-site blast radius.

## Scope

- In scope:
  - Implement `POST /api/environments/{id}/restore`.
  - Enqueue `env_restore` job and enforce restore state transitions.
  - Require backup reference validation.
  - Require a pre-restore full backup of the target environment.
- Out of scope:
  - Point-in-time database restore.
  - Partial table restore.

## Allowed Change Paths

- `internal/api/**`
- `internal/environments/**`
- `internal/jobs/**`
- `internal/backups/**`
- `ansible/playbooks/env-restore.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-environment-restore.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Restore endpoint validates `backup_id` and returns `202` with `job_id`.
2. Restore sets environment/site status to `restoring` and finalizes to `active` or `failed` in one transaction.
3. Restore operation scopes strictly to target environment resources.
4. Failed restore retains actionable error code/message on job record.
5. Restore creates or verifies a pre-restore full backup before applying backup content.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/env-restore.yml`
- Required tests:
  - Restore handler validation tests.
  - Restore state transition tests.

## Risks and Rollback

- Risk: restore to wrong environment due to ID mismatch.
- Rollback: enforce environment-backup ownership check before job enqueue.
