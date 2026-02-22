# Design: dashboard-runtime-realignment

## Architecture and Boundaries

- Services/components affected:
  - `internal/devserver` for route hierarchy and seed-data removal.
  - `internal/api` for `GET /api/nodes` and runtime inventory query endpoint.
  - `internal/ssh` and environment lookup path for WordPress version node query.
  - `internal/jobs`/runtime startup path for active worker loop in `pressluft dev`.
- Ownership boundaries:
  - Infrastructure mutations remain queue + Ansible only.
  - Runtime inventory is a synchronous node query (single SSH command, 10-second timeout).
  - No DB schema changes.
- Data flow summary:
  - Dashboard `/nodes` -> `GET /api/nodes`.
  - Dashboard `/sites` -> `GET /api/sites` + `GET /api/sites/{id}/environments` + targeted `GET /api/environments/{id}/wordpress-version`.
  - Runtime jobs in dev are executed by an in-process worker loop using existing handler map.

## Technical Plan

1. Finish Wave 5.5 smoke unblock by adding worker loop lifecycle into dev runtime startup/shutdown and verify queued `site_create` transitions.
2. Define and implement API contract additions for nodes list and WordPress version query, including traceability/error code docs.
3. Refactor dev dashboard route map to `/`, `/nodes`, `/sites`, `/jobs`; move environment/backups interactions under `/sites` only.
4. Remove hard-coded seed records and switch dashboard to real runtime state + explicit empty states.
5. Update plan/progress/dependency docs so `/resume-run` executes these tasks before Wave 6.

## Dependencies

- Depends on:
  - `changes/wp-first-runtime/` tasks through W5.5-T5 completion.
  - Existing node and environment stores.
- Blocks:
  - Wave 6 (`W6-T1` onward).

## Risks and Mitigations

- Risk: route changes can break deep links and tests.
  - Mitigation: update route tests first and keep strict route allowlist assertions.
- Risk: synchronous WordPress version query slows `/sites` rendering.
  - Mitigation: use bounded timeout and render per-row fallback/error states.
- Risk: worker loop integration can introduce goroutine lifecycle leaks.
  - Mitigation: explicit context cancellation and shutdown verification tests.

## Rollback

- Safe rollback sequence:
  1. Keep worker-loop fix if stable and isolate rollback to dashboard/API inventory changes.
  2. Revert dashboard route map to previous shell while preserving new tests as skipped pending rework.
  3. Remove new endpoints from OpenAPI/docs in same rollback PR to preserve contract consistency.
