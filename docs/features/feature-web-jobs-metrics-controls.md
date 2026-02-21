Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, contracts/openapi.yaml, docs/features/feature-jobs-and-metrics.md, docs/features/feature-job-control.md
Supersedes: none

# FEATURE: web-jobs-metrics-controls

## Problem

Operators need in-dashboard observability and administrative controls to trust async automation and recover from failed states quickly.

## Scope

- In scope:
  - Jobs list/detail UI.
  - Metrics snapshot visibility in dashboard shell.
  - Job cancel + site/environment reset actions with conflict-state handling.
- Out of scope:
  - Long-term analytics dashboards.
  - Prometheus export UI.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-jobs-metrics-controls.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Dashboard surfaces running/recent jobs and job details with error metadata.
2. Metrics counters are visible and refreshed without breaking authenticated shell flow.
3. Job cancel and reset actions surface success and deterministic `409` conflict errors.

## Verification

- Required commands:
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Jobs/metrics/control UI tests for authorized, unauthorized, and conflict states.

## Risks and Rollback

- Risk: admin controls can be exposed without adequate state guards in UX.
- Rollback: keep action buttons disabled unless server-permitted states are confirmed.
