Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/ui-flows.md, docs/features/feature-dashboard-site-centric-hierarchy.md, docs/features/feature-site-create.md, docs/features/feature-environment-create-clone.md, docs/features/feature-backups.md
Supersedes: none

# FEATURE: site-detail-drilldown

## Problem

The current `/sites` surface keeps environments and backups on the same page as the site index. Operators need a clearer site-first workflow where they pick a site first, then manage only that site's environments and backups in a dedicated details route.

## Scope

- In scope:
  - Add dedicated site details route `/sites/{site_id}` in dashboard shell routing.
  - Keep `/sites` as the site index/create surface.
  - Move environment and backup management UI to the dedicated site details route only.
  - Add a row-end three-point menu on site rows with quick actions (open details, create environment, create backup).
  - Ensure deep-link and browser refresh behavior works for `/sites/{site_id}`.
- Out of scope:
  - New API endpoints.
  - New infrastructure or queue behavior.
  - Nuxt migration work.

## Allowed Change Paths

- `internal/devserver/**`
- `docs/ui-flows.md`
- `docs/features/feature-site-detail-drilldown.md`
- `docs/features/README.md`
- `PLAN.md`
- `PROGRESS.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. `/sites` remains the index page and does not host environment/backups management panels.
2. `/sites/{site_id}` serves dashboard shell and shows environment/backups management scoped to that site.
3. Site table includes a three-point menu at row end with quick actions to open details and jump to environment/backup actions.
4. `/environments` and `/backups` remain unavailable as top-level routes.
5. `/resume-run` continues from this feature work before Wave 6 tasks.

## Wave 5.8 Stabilization Bug List

1. `BUG-SD-01` - Invalid nested site-detail paths like `/sites/<id>/...` are currently served by the dashboard shell instead of returning `404`.
2. `BUG-SD-02` - Invalid requested site detail IDs can still trigger create-form submissions and produce avoidable API round-trips.
3. `BUG-SD-03` - Site detail not-found state messaging is not treated as a submission gate for environment/backup create actions.

Wave 5.8 acceptance addendum:

1. Only `/sites/{site_id}` (single path segment) serves the site-detail shell; `/sites/` and nested `/sites/{site_id}/...` paths return `404`.
2. When `/sites/{site_id}` points at a missing site, environment and backup create forms must fail fast client-side with a deterministic `404 requested site not found` message.
3. Site detail not-found behavior remains visible in environment and backup tables without changing API contracts.

## Scenarios (WHEN/THEN)

1. WHEN an operator opens `/sites` THEN they can create/list sites and access per-site quick actions.
2. WHEN an operator chooses site details THEN the app navigates to `/sites/{site_id}` and shows only that site's environments/backups.
3. WHEN the operator refreshes `/sites/{site_id}` THEN the same site-scoped detail context is restored.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `go run ./cmd/pressluft dev --port 18600` and `curl http://127.0.0.1:18600/sites && curl http://127.0.0.1:18600/sites/test-site && curl http://127.0.0.1:18600/environments && curl http://127.0.0.1:18600/backups`
- Required tests:
  - `internal/devserver/server_test.go` route coverage for `/sites/{id}` and top-level removed routes.
  - `internal/devserver/server_test.go` marker assertions for site row quick actions and site detail panels.

## Risks and Rollback

- Risk: route split may regress existing environment/backup create flows.
- Rollback: restore combined `/sites` panel behavior while keeping `/nodes` and removed top-level routes unchanged.
