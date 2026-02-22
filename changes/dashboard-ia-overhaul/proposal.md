Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/features/feature-dashboard-ia-overhaul.md, docs/changes-workflow.md, PLAN.md
Supersedes: none

# Proposal: dashboard-ia-overhaul

## Problem

The embedded operator dashboard has grown as a single mixed surface, making workflows harder to navigate and increasing coupling between unrelated concerns (sites, environments, backups, jobs). This slows operator workflows and raises regression risk for upcoming Wave 6 work.

## Scope

- In scope:
  - Define and implement route-level dashboard subsites for concern separation.
  - Improve informational hierarchy and shared context visibility.
  - Refactor embedded dashboard code organization to reduce cross-concern coupling.
- Out of scope:
  - New API endpoints or contract changes.
  - Schema or migration changes.
  - Migration to a separate Nuxt `web/` frontend.

## Impact Summary

- API contract: none
- DB schema/migrations: none
- Infra/playbooks: none
- Security posture: none

## Governing Specs

- `docs/spec-index.md`
- `docs/ui-flows.md`
- `docs/technical-architecture.md`
- `docs/features/feature-dashboard-ia-overhaul.md`

## Acceptance Criteria

1. Operators can navigate concern-scoped dashboard subsites at stable routes.
2. Existing site/environment/backup/jobs workflows continue working with clearer hierarchy and reduced coupling.
3. Change packet tasks map directly to Wave 5 tasks W5-T4 through W5-T7 with verification gates defined before execution.
