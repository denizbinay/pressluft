Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/backups-and-restore.md, docs/job-execution.md, docs/job-types.md, docs/ansible-execution.md
Supersedes: none

# FEATURE: backup-retention-cleanup

## Problem

Operators need guaranteed enforcement of retention policy so expired backups are removed predictably.

## Scope

- In scope:
  - Implement scheduled `backup_cleanup` job execution.
  - Delete off-site backup objects for records with expired `retention_until`.
  - Mark backup records and cleanup results deterministically.
- Out of scope:
  - Tiered retention policies by site plan.
  - Non-S3 storage backends.

## Allowed Change Paths

- `internal/jobs/**`
- `internal/backups/**`
- `internal/store/**`
- `ansible/playbooks/backup-cleanup.yml`
- `docs/backups-and-restore.md`
- `docs/job-types.md`
- `docs/ansible-execution.md`
- `docs/features/feature-backup-retention-cleanup.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/backups-and-restore.md`
- `docs/job-types.md`
- `docs/ansible-execution.md`

## Acceptance Criteria

1. Expired backups are detected using `retention_until < now` and queued for cleanup.
2. `backup_cleanup` executes through job queue + Ansible only.
3. Successful cleanup removes remote objects and transitions backup state to `expired` where applicable.
4. Cleanup failures persist structured `jobs.error_code` and support retry policy.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/backup-cleanup.yml`
- Required tests:
  - Retention selection and enqueue tests.
  - Cleanup executor tests for success/failure/retry behavior.

## Risks and Rollback

- Risk: cleanup bug could delete non-expired backups.
- Rollback: disable scheduler path and restore deleted objects from object-store versioning where available.
