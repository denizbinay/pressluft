# Proposal: dashboard-runtime-realignment

## Problem

Wave 5 dashboard IA completion did not match operator intent and current runtime truth. The dashboard still over-emphasizes top-level environments/backups views, lacks a first-class nodes surface, and ships seed records that make operational state ambiguous. In parallel, Wave 5.5 smoke remains blocked because dev runtime does not process queued jobs end-to-end.

## Scope

- In scope:
  - Complete Wave 5.5 runtime smoke unblock by wiring active queue worker loop in `pressluft dev` path.
  - Realign dashboard hierarchy to node-first and site-centric views (`/`, `/nodes`, `/sites`, `/jobs`).
  - Add live runtime inventory APIs for nodes list and WordPress version lookup.
  - Remove seeded placeholder records from dev dashboard runtime.
  - Update plan/progress docs so `/resume-run` executes this correction before Wave 6.
- Out of scope:
  - Schema migrations.
  - Multi-node external fleet management execution.
  - Broad redesign of future Wave 6+ release workflows.

## Impact Summary

- API contract: update-required
- DB schema/migrations: none
- Infra/playbooks: none
- Security posture: update-required

## Governing Specs

- `docs/spec-index.md`
- `docs/technical-architecture.md`
- `docs/data-model.md`
- `docs/job-execution.md`
- `docs/security-and-secrets.md`
- `docs/features/feature-wp-first-runtime.md`
- `docs/features/feature-dashboard-site-centric-hierarchy.md`
- `docs/features/feature-runtime-inventory-queries.md`

## Acceptance Criteria

1. Wave 5.5 smoke (`create site -> reachable preview URL`) passes in local/self-node baseline and is documented as complete in plan/progress.
2. Dashboard hierarchy is node-first and site-centric (`/`, `/nodes`, `/sites`, `/jobs`) with no top-level environments/backups routes.
3. Operators can view all nodes and local-node readiness, and can see site rows with node placement, status, preview URL, and live WordPress version.
4. Runtime inventory endpoints and docs are contract-aligned and traceable to owning feature specs.
