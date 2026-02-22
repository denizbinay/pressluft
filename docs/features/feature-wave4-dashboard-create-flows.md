Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/ui-flows.md, docs/features/feature-site-create.md, docs/features/feature-environment-create-clone.md
Supersedes: none

# FEATURE: wave4-dashboard-create-flows

## Problem

Operators can create sites and environments through API endpoints, but Wave 4 is not complete until the dashboard exposes those flows and resulting state in a browser-visible path.

## Scope

- In scope:
  - Add dashboard UI create flow for sites.
  - Add dashboard UI create flow for environments (`staging` and `clone`).
  - Show site and environment state after create actions.
  - Surface API validation/conflict failures as inline operator feedback.
- Out of scope:
  - New API endpoints.
  - Database schema changes.
  - Migration from `internal/devserver` dashboard to `web/` Nuxt implementation.

## Allowed Change Paths

- `internal/devserver/**`
- `internal/api/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/store/**`
- `docs/ui-flows.md`
- `docs/features/feature-wave4-dashboard-create-flows.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

If update is required, list exact contract/spec files.

- `none`

## Acceptance Criteria

1. Authenticated operator can create a site from the dashboard using existing `POST /api/sites` behavior.
2. Authenticated operator can create an environment from the dashboard using existing `POST /api/sites/{id}/environments` behavior.
3. Dashboard shows site/environment records and statuses after create actions.
4. Dashboard renders contract-aligned errors for `400`, `404`, and `409` responses.
5. Wave 4 manual browser flow can be completed without direct API calls.

## Scenarios (WHEN/THEN)

1. WHEN the operator submits valid site create input THEN the dashboard shows accepted state and the new site appears in list/detail state.
2. WHEN the operator submits valid staging or clone environment input THEN the dashboard shows accepted state and the environment appears with expected status.
3. WHEN the operator submits invalid or conflicting input THEN the dashboard remains usable, preserves existing state, and shows an inline error message.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - `internal/devserver/server_test.go`
  - `internal/api/router_test.go`

## Risks and Rollback

- Risk: dashboard data mapping can diverge from API payload semantics.
- Rollback: revert `internal/devserver/**` create-flow UI changes and keep existing API surfaces unchanged.
