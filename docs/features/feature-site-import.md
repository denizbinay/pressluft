Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/migration-spec.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: site-import

## Problem

Existing WordPress projects need a deterministic import path into Pressluft-managed environments.

## Scope

- In scope:
  - Implement `POST /api/sites/{id}/import` async behavior.
  - Enqueue `site_import` job with `archive_url` payload.
  - Enforce serialized-safe URL rewrite workflow.
- Out of scope:
  - Direct cPanel/Plesk migration adapters.
  - Incremental sync migration.

## Allowed Change Paths

- `internal/api/**`
- `internal/jobs/**`
- `internal/migration/**`
- `internal/store/**`
- `ansible/playbooks/site-import.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/migration-spec.md`
- `docs/features/feature-site-import.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/migration-spec.md`

## Acceptance Criteria

1. Import endpoint validates payload and returns `202` with `job_id`.
2. Import failures produce structured job errors and retry behavior per job rules.
3. Successful import leaves environment in `active` with updated release reference.
4. Serialized URL rewrite result matches environment preview or primary domain target.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/site-import.yml`
- Required tests:
  - Import handler validation tests.
  - Job executor tests for site_import.

## Risks and Rollback

- Risk: import can produce broken URLs if rewrite step fails silently.
- Rollback: keep pre-import backup and revert environment symlink/database from backup.
