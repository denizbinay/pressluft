Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/backups-and-restore.md, docs/job-execution.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: wave5-backup-restore-vertical-slice

## Problem

Current Wave 5 backup execution is placeholder-only, and restore contract coverage is incomplete for real operator recovery workflows.

## Scope

- In scope:
  - Replace backup placeholder execution with real artifact creation and metadata integrity checks.
  - Implement end-to-end `env_restore` path (API/service/job/playbook) with environment-scoped safety semantics.
  - Enforce pre-restore full backup requirement before restore application.
  - Add dashboard site-detail restore flow and deterministic success/failure feedback.
- Out of scope:
  - Cross-site restore.
  - Point-in-time database restore.
  - Wave 6 health/rollback orchestration expansion.

## Allowed Change Paths

- `internal/backups/**`
- `internal/environments/**`
- `internal/api/**`
- `internal/jobs/**`
- `internal/store/**`
- `internal/devserver/**`
- `ansible/playbooks/backup-create.yml`
- `ansible/playbooks/env-restore.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/job-types.md`
- `docs/ui-flows.md`
- `docs/features/feature-wave5-backup-restore-vertical-slice.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/job-types.md`

## Acceptance Criteria

1. `backup_create` produces real artifacts and records checksum + size from actual output.
2. `POST /api/environments/{id}/restore` validates `backup_id` ownership, enqueues `env_restore`, and enforces pre-restore backup guard.
3. Restore execution affects only target environment resources and preserves site scoping invariants.
4. Backup/restore failures produce stable job errors with operator-actionable messages.
5. Site-detail UI supports backup select + restore confirmation and displays terminal outcomes.
6. End-to-end backup create/list/restore smoke passes, including post-restore preview reachability.

## Scenarios (WHEN/THEN)

1. WHEN operator creates backup THEN artifact metadata and status transitions reflect actual execution.
2. WHEN operator restores a valid backup THEN environment returns to active with reachable preview URL.
3. WHEN restore preconditions fail (invalid backup/scope/checksum/pre-restore backup failure) THEN restore does not proceed and returns stable failure semantics.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/backup-create.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/env-restore.yml`
  - `bash scripts/smoke-backup-restore.sh`
- Required tests:
  - Backup execution metadata integrity tests.
  - Restore handler/service state transition tests.
  - Site-detail restore flow regression coverage.

## Risks and Rollback

- Risk: restore can leave runtime partially updated on mid-run failures.
- Rollback: enforce pre-restore checkpoint backup and restore-on-failure fallback path before marking terminal failure.
