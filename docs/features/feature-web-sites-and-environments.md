Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, contracts/openapi.yaml, docs/contract-traceability.md
Supersedes: none

# FEATURE: web-sites-and-environments

## Problem

Operators need first-class dashboard flows for site and environment discovery/creation to make core lifecycle automation usable.

## Scope

- In scope:
  - Sites list/detail/create dashboard flows.
  - Environment list/detail/create (staging/clone) dashboard flows.
  - UX state wiring for async creation jobs.
- Out of scope:
  - Deploy/update/restore/promote flows.
  - Domain, backup, cache, and magic-login controls.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-sites-and-environments.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Users can create a site from dashboard and observe async job progression to completion.
2. Users can create staging/clone environments with validated payload controls.
3. Sites and environments list/detail views refresh deterministically after job completion.

## Verification

- Required commands:
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Sites and environments flow tests covering success and validation-error states.

## Risks and Rollback

- Risk: stale list refresh behavior can hide completed resources.
- Rollback: centralize post-job refresh path and preserve server truth over optimistic local state.
