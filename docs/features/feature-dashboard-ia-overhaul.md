Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/ui-flows.md, docs/technical-architecture.md, docs/features/feature-wave4-dashboard-create-flows.md, docs/features/feature-backups.md, docs/features/feature-jobs-and-metrics.md
Supersedes: none

# FEATURE: dashboard-ia-overhaul

## Problem

The embedded operator dashboard currently presents auth, metrics, site creation, environment creation, backups, and jobs in one mixed surface. Operators cannot navigate by concern cleanly, and state coupling between panels makes workflows harder to understand and extend.

## Scope

- In scope:
  - Introduce route-level dashboard subsites for operator concerns (`/`, `/sites`, `/environments`, `/backups`, `/jobs`).
  - Separate view, state, and data-fetch concerns inside `internal/devserver` so each subsite owns its own UI behavior.
  - Preserve existing API behavior while improving information hierarchy and workflow clarity.
  - Maintain desktop/mobile usability for the embedded dashboard.
- Out of scope:
  - New API endpoints or response shape changes.
  - Migration from embedded dashboard to a standalone `web/` app.
  - Backend lifecycle behavior changes for sites, environments, backups, or jobs.

## Allowed Change Paths

- `internal/devserver/**`
- `internal/api/**`
- `PLAN.md`
- `PROGRESS.md`
- `docs/ui-flows.md`
- `docs/features/feature-dashboard-ia-overhaul.md`
- `docs/features/README.md`
- `changes/dashboard-ia-overhaul/**`
- `changes/README.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

If update is required, list exact contract/spec files.

- `none`

## Acceptance Criteria

1. Authenticated operators can navigate between dedicated dashboard subsites for overview, sites, environments, backups, and jobs using stable URL routes.
2. Each subsite shows only concern-relevant information and actions, with shared context (selected site/environment) clearly surfaced where needed.
3. Existing create/list flows for sites, environments, backups, and jobs timeline remain functional and contract-aligned (`400`/`404`/`409` error rendering preserved).
4. Dashboard code structure in `internal/devserver` reflects clear separation of concerns (routing/shell, per-subsite rendering/handlers, shared API/state utilities).
5. `/resume-run` chooses this overhaul work before Wave 6 tasks by reading updated plan/progress state.

## Scenarios (WHEN/THEN)

1. WHEN an authenticated operator opens `/sites` THEN site-focused actions and lists are shown without unrelated backup/job control noise.
2. WHEN an operator switches to `/backups` with no environment selected THEN the UI presents a clear context requirement and no hidden coupling error.
3. WHEN an operator deep-links to `/jobs` THEN jobs list and timeline render without first visiting another dashboard section.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `go run ./cmd/pressluft dev --port 18180` and `curl http://127.0.0.1:18180/ && curl http://127.0.0.1:18180/sites && curl http://127.0.0.1:18180/environments && curl http://127.0.0.1:18180/backups && curl http://127.0.0.1:18180/jobs`
- Required tests:
  - `internal/devserver/server_test.go`

## Risks and Rollback

- Risk: route and shared-context refactor can break existing create/list flows.
- Rollback: revert `internal/devserver/**` to pre-overhaul dashboard shell and keep existing API services unchanged.
