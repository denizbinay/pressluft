Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/spec-index.md, docs/security-and-secrets.md, docs/api-contract.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: auth-session

## Problem

Operators need stable, secure control-plane authentication with predictable session behavior.

## Scope

- In scope:
  - Implement `POST /api/login` and `POST /api/logout`.
  - Enforce session token creation, expiry, and revocation behavior.
  - Return structured auth errors.
- Out of scope:
  - Multi-user roles.
  - Password reset and MFA.

## Allowed Change Paths

- `internal/api/**`
- `internal/auth/**`
- `internal/store/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/security-and-secrets.md`
- `docs/features/feature-auth-session.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Valid credentials return `200`, create an auth session, and set `session_token` as an HTTP cookie.
2. Invalid credentials return `401` with stable error payload shape.
3. Logout returns `200`, revokes the current session token, and clears the auth cookie.
4. Protected endpoints reject expired or revoked tokens with `401`.

## Scenarios (WHEN/THEN)

1. WHEN valid credentials are submitted to `POST /api/login` THEN the API returns `200` and sets `session_token` cookie.
2. WHEN invalid credentials are submitted to `POST /api/login` THEN the API returns `401` with the canonical error shape.
3. WHEN a revoked or expired session token is used on a protected endpoint THEN access is denied with `401`.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Login success/failure handler tests.
  - Session expiry and revocation tests.

## Risks and Rollback

- Risk: Session invalidation bugs can lock out admins.
- Rollback: revert auth route changes and restore previous session validation behavior.
