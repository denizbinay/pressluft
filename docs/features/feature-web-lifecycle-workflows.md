Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/ui-flows.md, contracts/openapi.yaml, docs/contract-traceability.md, docs/health-checks.md, docs/promotion-and-drift.md
Supersedes: none

# FEATURE: web-lifecycle-workflows

## Problem

MVP requires safe lifecycle mutations in dashboard UX so operators can deploy, update, restore, and promote with explicit guardrails.

## Scope

- In scope:
  - Dashboard mutation flows for deploy, updates, restore, drift check, and promote.
  - Job progress and terminal-state handling in UI.
  - Explicit guardrail messaging for drift and backup requirements.
- Out of scope:
  - Backup list/create views (handled in operations workflows).
  - Job administration pages.

## Allowed Change Paths

- `web/**`
- `docs/features/feature-web-lifecycle-workflows.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Deploy and updates can be initiated from environment UI with deterministic form validation.
2. Restore and promote flows enforce existing API guardrails and display blocking errors clearly.
3. Drift-check and health-result states are visible where users make promotion and release decisions.

## Verification

- Required commands:
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - Lifecycle workflow tests for success, conflict, and validation error states.

## Risks and Rollback

- Risk: missing guardrail cues can lead to incorrect operator actions.
- Rollback: disable affected action paths in UI until blocking-state rendering is corrected.
