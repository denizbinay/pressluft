Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, contracts/openapi.yaml, docs/contract-traceability.md
Supersedes: none

# FEATURE: web-operations-workflows

## Problem

Operators need dashboard controls for backups, domains, caching, and magic login to complete routine WordPress operations without dropping to API tooling.

## Scope

- In scope:
  - Backup create/list workflow UI.
  - Domain add/remove/list and TLS status UI.
  - Cache toggle/purge controls.
  - Magic login action and synchronous error handling.
- Out of scope:
  - Job administration screens.
  - Internal settings admin UI.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-operations-workflows.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Operators can create backups and see retention/status fields in environment context.
2. Operators can add/remove domains and see TLS lifecycle status updates.
3. Operators can toggle/purge cache and invoke magic login with deterministic success/error feedback.

## Verification

- Required commands:
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Operations workflow tests covering success, conflict, and node-query error scenarios.

## Risks and Rollback

- Risk: synchronous magic-login error states may be misreported as job failures.
- Rollback: preserve dedicated synchronous handling path with explicit error code mapping in UI.
