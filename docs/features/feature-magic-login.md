Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/security-and-secrets.md, docs/technical-architecture.md, docs/api-contract.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: magic-login

## Problem

Pressluft operators need immediate WordPress admin access per environment without storing or manually managing WordPress credentials.

## Scope

- In scope:
  - Implement `POST /api/environments/:id/magic-login` as a synchronous node query.
  - Validate environment exists and is in `active` status before execution.
  - Run single SSH command with a 10-second hard timeout.
  - Return `{ login_url, expires_at }` on success.
  - Return structured error codes for known failure classes.
  - Add audit log entry for each request.
- Out of scope:
  - New authentication models.
  - Multi-user role support.
  - Any asynchronous job-queue implementation for this endpoint.
  - Changes to WordPress core/session internals beyond token generation script execution.

## Allowed Change Paths

- `internal/api/**`
- `internal/ssh/**`
- `internal/jobs/**`
- `internal/store/**`
- `internal/audit/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/security-and-secrets.md`
- `docs/ui-flows.md`
- `docs/features/feature-magic-login.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. `POST /api/environments/:id/magic-login` returns `200` with `login_url` and `expires_at` when environment status is `active` and node query succeeds.
2. Endpoint returns `409` with `code = environment_not_active` when environment status is not `active`.
3. Endpoint returns `502` with `code = node_unreachable` on SSH timeout/connection failure, and `502` with `code = wp_cli_error` when WP-CLI command fails.
4. Endpoint does not enqueue a job and does not return `job_id`.
5. API writes an audit log entry with `action = magic_login`, `resource_type = environment`, and `resource_id = {environment_id}`.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - API handler tests for success and all defined error codes.
  - SSH command timeout behavior test (10-second cap).
  - Audit log persistence test for success and failure paths.

## Risks and Rollback

- Risk: SSH execution may leak low-level errors directly to clients.
- Rollback: disable endpoint route and return a stable `501` until corrected; keep OpenAPI and docs updated in same change.
