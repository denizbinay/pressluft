Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/features/feature-dashboard-ia-overhaul.md, docs/ui-flows.md, docs/technical-architecture.md
Supersedes: none

# Design: dashboard-ia-overhaul

## Architecture and Boundaries

- Services/components affected:
  - `internal/devserver` dashboard HTTP shell and embedded UI assets.
  - Existing API handlers only as consumers; no API behavior changes.
- Ownership boundaries:
  - Dashboard route shell owns top-level navigation.
  - Each subsite owns rendering/actions for one operator concern.
  - Shared context module owns selected site/environment state.
- Data flow summary:
  - View-specific loaders call existing `/api/*` endpoints.
  - Shared context emits state changes consumed by subsite modules.
  - Errors remain contract-aligned and scoped to active view.

## Technical Plan

1. Add a route-aware dashboard shell serving `/`, `/sites`, `/environments`, `/backups`, `/jobs`.
2. Split UI logic by concern with a minimal shared API/state utility layer.
3. Migrate existing create/list/timeline behavior into subsite modules without contract changes.
4. Extend tests to assert route markers and concern-scoped shell behavior.

## Delivery Slices

1. W5-T4: route-level shell and concern navigation (`/`, `/sites`, `/environments`, `/backups`, `/jobs`).
2. W5-T5: shared state/data flow refactor with explicit concern boundaries.
3. W5-T6: migrate existing forms/lists/timeline into route-scoped subsites.
4. W5-T7: extend dashboard tests and run verification gates before Wave 6.

## Dependencies

- Depends on:
  - `docs/features/feature-dashboard-ia-overhaul.md`
  - Existing Wave 4 and Wave 5 feature behavior already implemented
- Blocks:
  - Wave 6 task start in `PLAN.md` (`W6-T1` now depends on `W5-T7`)

## Risks and Mitigations

- Risk: route migration can break direct `/` startup flow.
  - Mitigation: preserve `/` as default overview route and add route smoke checks.
- Risk: shared context bugs can mis-scope environment/backup actions.
  - Mitigation: explicit context-empty states and targeted regression tests.

## Rollback

- Safe rollback sequence:
  1. Revert `internal/devserver/**` to pre-overhaul shell.
  2. Keep API/backend unchanged.
  3. Re-run backend gates and dashboard shell smoke checks.
