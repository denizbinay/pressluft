Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/backups-and-restore.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: backups

## Problem

Operators need reliable backup creation and visibility before risky operations.

## Scope

- In scope:
  - Implement `POST /api/environments/:id/backups` and `GET /api/environments/:id/backups`.
  - Enqueue `backup_create` jobs and reflect backup lifecycle states.
- Out of scope:
  - Cross-project backup federation.
  - Non-S3 backup providers for MVP.

## Allowed Change Paths

- `internal/api/**`
- `internal/backups/**`
- `internal/jobs/**`
- `internal/store/**`
- `ansible/playbooks/backup-create.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-backups.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Backup create endpoint validates `backup_scope` and returns `202` with `job_id`.
2. Backup list endpoint returns stateful records with retention metadata.
3. Backup status transitions follow `pending -> running -> completed|failed|expired`.
4. Backup failures surface structured job errors.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/backup-create.yml`
- Required tests:
  - Backup create/list handler tests.
  - Backup lifecycle transition tests.

## Risks and Rollback

- Risk: storage upload failure can mark backup success incorrectly.
- Rollback: enforce checksum/complete marker validation before `completed` state.
