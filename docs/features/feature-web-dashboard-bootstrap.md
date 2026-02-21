Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, contracts/openapi.yaml, docs/api-contract.md, docs/testing.md
Supersedes: none

# FEATURE: web-dashboard-bootstrap

## Problem

The MVP requires a complete Nuxt dashboard, but no `web/` workspace exists, so UI flows cannot be implemented or verified.

## Scope

- In scope:
  - Create the Nuxt 3 + Vue 3 + TypeScript dashboard workspace under `web/`.
  - Establish typed API client foundation mapped to existing OpenAPI operations.
  - Add base lint/build scripts required by frontend gates.
- Out of scope:
  - Full workflow UI implementation.
  - Go-side embed/static serving.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-dashboard-bootstrap.md`
- `docs/testing.md`
- `PLAN.md`
- `PROGRESS.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. A runnable Nuxt dashboard workspace exists under `web/` with TypeScript and `<script setup lang="ts">` conventions.
2. A typed API client layer is available for core authenticated API calls.
3. Frontend gates run successfully from repo root (`pnpm install`, `pnpm lint`, `pnpm build` from `web/`).

## Verification

- Required commands:
  - `cd web && pnpm install`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Typecheck/lint coverage for app bootstrap and API client modules.

## Risks and Rollback

- Risk: client scaffolding drifts from OpenAPI naming and causes integration churn.
- Rollback: keep API client surface minimal and regenerate/realign before feature-layer UI work.
