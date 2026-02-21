Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/testing.md, docs/ui-flows.md, docs/changes-workflow.md
Supersedes: none

# FEATURE: web-dashboard-hardening

## Problem

Before MVP release, dashboard behavior must be deterministic under async/error conditions and protected by CI gates.

## Scope

- In scope:
  - Enforce frontend CI gates for lint/build.
  - Harden async loading, error handling, and basic accessibility behavior.
  - Run MVP release-readiness smoke checks for embedded dashboard delivery.
- Out of scope:
  - New API endpoints.
  - Non-MVP analytics and advanced dashboard customization.

## Allowed Change Paths

- `web/**`
- `.github/workflows/ci.yml`
- `docs/testing.md`
- `docs/features/feature-web-dashboard-hardening.md`
- `PLAN.md`
- `PROGRESS.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. CI runs and enforces frontend gates whenever `web/` is present.
2. Key dashboard workflows expose deterministic loading, empty, success, and error states.
3. Embedded dashboard smoke checks pass in release-readiness verification.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `cd web && pnpm install`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Dashboard smoke path tests for login and at least one async mutation flow.

## Risks and Rollback

- Risk: brittle UI state handling can cause release regressions despite passing API tests.
- Rollback: keep hardening changes scoped and revert workflow-specific regressions without changing API behavior.
