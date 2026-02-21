Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, docs/features/feature-auth-session.md, contracts/openapi.yaml, docs/api-contract.md
Supersedes: none

# FEATURE: web-auth-and-shell

## Problem

Dashboard users need an authenticated web entrypoint and protected navigation shell that matches cookie-session API behavior.

## Scope

- In scope:
  - Implement web login/logout flow against `POST /api/login` and `POST /api/logout`.
  - Add protected app shell routes with auth guard behavior.
  - Implement unauthorized handling and redirect to login.
- Out of scope:
  - Workflow-specific pages beyond shell placeholders.
  - Multi-user auth/roles.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-auth-and-shell.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Valid login creates an authenticated dashboard session and navigates to the protected shell.
2. Logout clears session state and returns user to login.
3. Protected routes handle `401` deterministically by redirecting to login.

## Verification

- Required commands:
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Login success/failure UI tests.
  - Route guard/auth redirect tests.

## Risks and Rollback

- Risk: cookie/session handling mismatch can cause redirect loops.
- Rollback: reduce guard complexity and centralize session check strategy in one middleware path.
