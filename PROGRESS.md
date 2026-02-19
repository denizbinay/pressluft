# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-20

## Current Stage

- Stage: pre-implementation readiness complete.
- Planning artifacts initialized: `PLAN.md`, `PROGRESS.md`.

## Completed

- Baseline readiness audit executed against `docs/pre-plan-readiness.md`.
- OpenAPI/traceability parity validated.
- Job type registry/ownership parity validated.
- Baseline schema migration directory and migration runner created.
- Contract drift fixed for global jobs (`JobStatusResponse.site_id` nullable).

## In Progress

- None.

## Next Up

1. Iteration 0 bootstrap: create runnable Go module structure (`cmd/pressluft`, `internal/**`).
2. Add minimal migration verification in CI/local scripts.
3. Start Iteration 1 (auth + job queue foundations).

## Open Risks

- Build/test gates in docs are not yet runnable until Go project scaffolding exists.
- Migration runner depends on `sqlite3` CLI availability in the environment.
