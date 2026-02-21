# Pressluft MVP Plan

Status: active
Owner: platform
Last Updated: 2026-02-21

## Preconditions (Must Stay Green)

1. Spec/contract readiness checks pass: `bash scripts/check-readiness.sh`.
2. Baseline executable schema exists under `migrations/` and applies successfully.
3. Contract authority remains OpenAPI-first (`contracts/openapi.yaml`).

## Wave Map

- Wave G: governance and spec-routing hardening baseline.
- Wave 0: repository bootstrap and verification baseline.
- Wave 1: core runtime and mutation queue foundations.
- Wave 2: site/environment creation with backup and health foundations.
- Wave 3: deploy/update/restore/promotion safety.
- Wave 4: domains/cache/operator workflows.
- Wave 5: observability/control surfaces.
- Wave 6: dashboard foundation and Go embed delivery.
- Wave 7: dashboard workflow completion for all MVP UI flows.
- Wave 8: dashboard hardening, CI frontend gates, and MVP release readiness.

Merge points:

- MP1: end of Wave 1 before lifecycle work.
- MP2: end of Wave 3 before domain/cache.
- MP3: end of Wave 5 before MVP release hardening.
- MP4: end of Wave 6 before workflow-complete dashboard surfaces.
- MP5: end of Wave 7 before dashboard hardening/release readiness.

## Atomic Task Backlog

### Wave G - Governance and Routing Hardening (Completed)

- [x] WG-T1: Add top-level routing specs for discoverability (`SPEC.md`, `ARCHITECTURE.md`, `CONTRACTS.md`).
  - Depends on: none
  - Governing docs: `docs/spec-index.md`, `AGENTS.md`
- [x] WG-T2: Introduce ADR system with template and first accepted record.
  - Depends on: WG-T1
  - Governing docs: `docs/spec-lifecycle.md`, `docs/changes-workflow.md`
- [x] WG-T3: Enforce parallel lock registry and stale-lock lint in readiness checks.
  - Depends on: WG-T1
  - Governing docs: `docs/parallel-execution.md`, `docs/agent-governance.md`, `docs/testing.md`
- [x] WG-T4: Tighten session start contract and scenario-level spec structure.
  - Depends on: WG-T1
  - Governing docs: `AGENTS.md`, `docs/templates/feature-spec-template.md`

### Wave 0 - Bootstrap and Gates

- [x] W0-T1: Establish runnable Go module layout (`cmd/pressluft`, `internal/**`).
  - Depends on: none
  - Feature spec: `docs/features/feature-install-bootstrap.md`
- [x] W0-T2: Ensure migration runner works on clean DB.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-install-bootstrap.md`
- [x] W0-T3: Make build/vet/test gates executable in CI.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-install-bootstrap.md`

### Wave 1 - Core Runtime and Queue

- [x] W1-T1: Implement node provisioning job contract and execution path.
  - Depends on: W0-T1, W0-T2
  - Feature spec: `docs/features/feature-node-provision.md`
- [x] W1-T2: Implement auth session login/logout + cookie lifecycle.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-auth-session.md`
- [x] W1-T3: Implement mutation queue locking invariants (site/node single-mutation).
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-node-provision.md`
- [x] W1-T4: Implement baseline audit logging for all mutating API actions.
  - Depends on: W1-T2
  - Feature spec: `docs/features/feature-audit-logging.md`

### Wave 2 - Site, Environment, and Backup Foundations

- [x] W2-T1: Implement site create API + `site_create` job.
  - Depends on: MP1
  - Feature spec: `docs/features/feature-site-create.md`
- [x] W2-T2: Implement environment create/clone API + `env_create` job.
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-environment-create-clone.md`
- [x] W2-T3: Implement backup create/list with retention surface.
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-backups.md`
- [x] W2-T4: Implement health checks and rollback (`health_check`, `release_rollback`).
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-health-checks-and-rollback.md`

### Wave 3 - Deployment, Restore, and Promotion Safety

- [x] W3-T1: Implement deploy/update mutations (`env_deploy`, `env_update`).
  - Depends on: W2-T3, W2-T4
  - Feature spec: `docs/features/feature-environment-deploy-updates.md`
- [x] W3-T2: Implement environment restore (`env_restore`) with safety checks.
  - Depends on: W3-T1, W2-T3
  - Feature spec: `docs/features/feature-environment-restore.md`
- [x] W3-T3: Implement drift check and promotion guardrails (`drift_check`, `env_promote`).
  - Depends on: W3-T1, W2-T3
  - Feature spec: `docs/features/feature-promotion-drift.md`

### Wave 4 - Domain, Cache, Operator Workflows

- [x] W4-T1: Implement domain add/remove and TLS status surfaces.
  - Depends on: MP2
  - Feature spec: `docs/features/feature-domains-and-tls.md`
- [x] W4-T2: Implement cache toggle/purge mutations.
  - Depends on: MP2
  - Feature spec: `docs/features/feature-cache-controls.md`
- [x] W4-T3: Implement magic login synchronous node query.
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-magic-login.md`
- [x] W4-T4: Implement site import flow (`site_import`).
  - Depends on: MP2
  - Feature spec: `docs/features/feature-site-import.md`
- [x] W4-T5: Implement settings domain config control surface.
  - Depends on: W4-T1
  - Feature spec: `docs/features/feature-settings-domain-config.md`

### Wave 5 - Observability and Control

- [x] W5-T1: Implement jobs and metrics read APIs.
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-jobs-and-metrics.md`
- [x] W5-T3: Implement backup retention cleanup orchestration.
  - Depends on: W2-T3
  - Feature spec: `docs/features/feature-backup-retention-cleanup.md`
- [x] W5-T4: Implement administrative job control actions.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-job-control.md`

### Wave 6 - Dashboard Foundation and Embed

- [x] W6-T1: Bootstrap Nuxt dashboard workspace and typed API client foundation.
  - Depends on: MP3
  - Feature spec: `docs/features/feature-web-dashboard-bootstrap.md`
- [x] W6-T2: Implement web auth/session flow and protected app shell.
  - Depends on: W6-T1
  - Feature spec: `docs/features/feature-web-auth-and-shell.md`
- [x] W6-T3: Serve built dashboard through the Go control plane with SPA fallback.
  - Depends on: W6-T1, W6-T2
  - Feature spec: `docs/features/feature-web-embed-and-delivery.md`


### Wave 7 - Dashboard Workflow Completion

- [x] W7-T1: Implement sites and environments dashboard flows.
  - Depends on: MP4
  - Feature spec: `docs/features/feature-web-sites-and-environments.md`
- [x] W7-T2: Implement deploy/update/restore/promote workflow UI with job progress wiring.
  - Depends on: W7-T1
  - Feature spec: `docs/features/feature-web-lifecycle-workflows.md`
- [x] W7-T3: Implement backups, domains, caching, and magic-login workflow UI.
  - Depends on: W7-T1
  - Feature spec: `docs/features/feature-web-operations-workflows.md`
- [x] W7-T4: Implement jobs/metrics visibility and administrative control UI.
  - Depends on: W7-T2, W7-T3
  - Feature spec: `docs/features/feature-web-jobs-metrics-controls.md`

### Wave 8 - Dashboard Hardening and MVP Release Readiness

- [x] W8-T1: Add frontend CI gates and deterministic command presets for dashboard verification.
  - Depends on: MP5
  - Feature spec: `docs/features/feature-web-dashboard-hardening.md`
- [x] W8-T2: Complete dashboard UX hardening for async states, error handling, and accessibility baselines.
  - Depends on: W8-T1
  - Feature spec: `docs/features/feature-web-dashboard-hardening.md`
- [x] W8-T3: Execute MVP release readiness pass with embedded dashboard smoke verification.
  - Depends on: W8-T2
  - Feature spec: `docs/features/feature-web-dashboard-hardening.md`

## Dependency and Contract Rules

- `contracts/openapi.yaml` is canonical for API behavior.
- Endpoint ownership remains 1:1 in `docs/contract-traceability.md`.
- Async endpoint job types must exist in `docs/job-types.md`.
- Error surfaces must be registered in `docs/error-codes.md`.
- Enum/state values must stay aligned with `docs/data-model.md` and `docs/state-machines.md`.

## Verification Gates

- Before commit:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
- Before PR:
  - `go test ./internal/... -v`
- Spec/contract checks:
  - `bash scripts/check-readiness.sh`
- Frontend gate (when `web/` exists):
  - `cd web && pnpm install`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`

## Planning Rules

- No implementation starts without its owning feature spec.
- No schema change ships without a migration.
- No API behavior change ships without OpenAPI update in the same change.
- Major changes must use `changes/<slug>/` workflow (`docs/changes-workflow.md`).
- No-forward-dependency proof is maintained in `docs/plan-dependency-matrix.md`.
