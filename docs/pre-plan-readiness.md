Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/contract-guardrails.md, docs/contract-traceability.md, docs/schema-authority.md, docs/features/README.md, docs/error-codes.md, docs/job-types.md, docs/testing.md
Supersedes: none

# Pre-Plan Readiness

This checklist must pass before creating `PLAN.md` and `PROGRESS.md`.

## Readiness Checklist

- [x] Metadata headers exist on all active docs.
- [x] No contradictory active specs across architecture/data/contracts/security.
- [x] Every OpenAPI endpoint has exactly one owning feature spec.
- [x] Contract guardrails are documented and accepted.
- [x] API and job error code registry exists and is referenced.
- [x] Canonical job type registry exists and is referenced.
- [x] Every canonical job type has an owning feature spec.
- [x] Schema authority chain is documented and accepted.
- [x] Baseline executable schema exists under `migrations/` and matches `docs/data-model.md`.
- [x] Feature specs include acceptance criteria, verification, and rollback.
- [x] Top-level spec routers (`SPEC.md`, `ARCHITECTURE.md`, `CONTRACTS.md`) exist and point to canonical docs.
- [x] ADR directory exists with template and at least one accepted decision record.
- [x] Parallel lock registry (`coordination/locks.md`) exists and passes lock lint checks.
- [x] Verification expectations are documented in `docs/testing.md`.

## Handoff Output

When all items are checked, planning handoff must include:

1. Prioritized implementation order by feature spec.
2. Dependency mapping for schema/contracts/tests.
3. Iteration structure for `PLAN.md` and `PROGRESS.md`.

## Automation

Run readiness checks with:

- `bash scripts/check-readiness.sh`

## Normalization Tracking

| Issue | Canonical Owner | Decision | Status |
|---|---|---|---|
| Auth transport ambiguity (cookie vs response token) | `contracts/openapi.yaml` | Cookie session is canonical; login/logout behavior expressed via cookie semantics, not bearer-style response token ownership | resolved |
| Traceability path mismatch (`/api` note vs table rows) | `docs/contract-traceability.md` | Matrix rows use full runtime paths with `/api` prefix | resolved |
| ACME client inconsistency | `docs/provisioning-spec.md` | certbot is the only supported ACME client for MVP | resolved |
| Job model mismatch for global jobs (`node_provision`) | `docs/data-model.md` + `docs/job-types.md` | `jobs.site_id` is nullable; global jobs require `site_id = NULL` and `node_id` | resolved |
| Promotion override policy conflict | `docs/promotion-and-drift.md` | Drift/backup gates are hard blocks with no admin override in MVP | resolved |
| Settings API contract mismatch | `contracts/openapi.yaml` + `docs/features/feature-settings-domain-config.md` | No public `/settings` API in MVP contract; settings behavior remains a non-public control-plane configuration surface | resolved |
| Environment isolation wording mismatch | `docs/technical-architecture.md` + `docs/user-requirements-and-workflows.md` | Isolation is per-environment runtime under site-keyed filesystem roots | resolved |
| Circular metadata dependencies | Doc headers in core/contract docs | Cycles removed from `Depends On` metadata where ownership was ambiguous | resolved |
| Job API payload/global job scope mismatch (`jobs.site_id` nullable vs API shape) | `contracts/openapi.yaml` + `docs/data-model.md` + `docs/api-contract.md` | `JobStatusResponse.site_id` allows null to represent global jobs such as `node_provision` | resolved |
