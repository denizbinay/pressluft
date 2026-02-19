Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/domain-and-routing.md, docs/provisioning-spec.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: cache-controls

## Problem

Operators need explicit control over FastCGI and Redis caches per environment.

## Scope

- In scope:
  - Implement `PATCH /api/environments/:id/cache` and `POST /api/environments/:id/cache/purge`.
  - Enqueue `env_cache_toggle` and `cache_purge` jobs.
  - Enforce validation that at least one cache toggle field is provided.
- Out of scope:
  - Global cache policy management across all environments.
  - Third-party CDN integration.

## Allowed Change Paths

- `internal/api/**`
- `internal/environments/**`
- `internal/jobs/**`
- `internal/store/**`
- `ansible/playbooks/env-cache-toggle.yml`
- `ansible/playbooks/cache-purge.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-cache-controls.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Cache toggle endpoint returns `202` with `job_id` and updates requested flags only.
2. Empty toggle payload returns `400`.
3. Cache purge endpoint returns `202` with `job_id`.
4. Jobs trigger corresponding Nginx/Redis actions and retain structured errors on failure.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/env-cache-toggle.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/cache-purge.yml`
- Required tests:
  - Cache toggle/purge handler tests.
  - Payload validation and job enqueue tests.

## Risks and Rollback

- Risk: toggle job may desync DB flags and runtime config.
- Rollback: regenerate env runtime config from DB and rerun cache toggle job.
