Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/contract-guardrails.md, docs/contract-traceability.md, docs/schema-authority.md, docs/features/README.md, docs/error-codes.md, docs/job-types.md
Supersedes: none

# Pre-Plan Readiness

This checklist must pass before creating `PLAN.md` and `PROGRESS.md`.

## Readiness Checklist

- [x] Metadata headers exist on all active docs.
- [ ] No contradictory active specs across architecture/data/contracts/security.
- [x] Every OpenAPI endpoint has exactly one owning feature spec.
- [x] Contract guardrails are documented and accepted.
- [x] API and job error code registry exists and is referenced.
- [x] Canonical job type registry exists and is referenced.
- [x] Every canonical job type has an owning feature spec.
- [x] Schema authority chain is documented and accepted.
- [ ] Baseline executable schema exists under `migrations/` and matches `docs/data-model.md`.
- [x] Feature specs include acceptance criteria, verification, and rollback.
- [x] Verification expectations are documented in `docs/testing.md`.

## Handoff Output

When all items are checked, planning handoff must include:

1. Prioritized implementation order by feature spec.
2. Dependency mapping for schema/contracts/tests.
3. Iteration structure for `PLAN.md` and `PROGRESS.md`.
