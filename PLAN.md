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
- Wave 2: site/environment lifecycle mutations.
- Wave 3: backup/restore/promotion safety.
- Wave 4: domains/cache/operator workflows.
- Wave 5: observability/control surfaces.

Merge points:

- MP1: end of Wave 1 before lifecycle work.
- MP2: end of Wave 3 before domain/cache.
- MP3: end of Wave 5 before MVP release hardening.

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

- [ ] W0-T1: Establish runnable Go module layout (`cmd/pressluft`, `internal/**`).
  - Depends on: none
  - Feature spec: `docs/features/feature-install-bootstrap.md`
- [ ] W0-T2: Ensure migration runner works on clean DB.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-install-bootstrap.md`
- [ ] W0-T3: Make build/vet/test gates executable in CI.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-install-bootstrap.md`

### Wave 1 - Core Runtime and Queue

- [ ] W1-T1: Implement node provisioning job contract and execution path.
  - Depends on: W0-T1, W0-T2
  - Feature spec: `docs/features/feature-node-provision.md`
- [ ] W1-T2: Implement auth session login/logout + cookie lifecycle.
  - Depends on: W0-T1
  - Feature spec: `docs/features/feature-auth-session.md`
- [ ] W1-T3: Implement mutation queue locking invariants (site/node single-mutation).
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-node-provision.md`

### Wave 2 - Site and Environment Lifecycle

- [ ] W2-T1: Implement site create API + `site_create` job.
  - Depends on: MP1
  - Feature spec: `docs/features/feature-site-create.md`
- [ ] W2-T2: Implement environment create/clone API + `env_create` job.
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-environment-create-clone.md`
- [ ] W2-T3: Implement deploy/update mutations (`env_deploy`, `env_update`).
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-environment-deploy-updates.md`
- [ ] W2-T4: Implement health checks and rollback (`health_check`, `release_rollback`).
  - Depends on: W2-T3
  - Feature spec: `docs/features/feature-health-checks-and-rollback.md`

### Wave 3 - Backup, Restore, Promotion

- [ ] W3-T1: Implement backup create/list with retention surface.
  - Depends on: W2-T4
  - Feature spec: `docs/features/feature-backups.md`
- [ ] W3-T2: Implement environment restore (`env_restore`) with safety checks.
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-environment-restore.md`
- [ ] W3-T3: Implement drift check and promotion guardrails (`drift_check`, `env_promote`).
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-promotion-drift.md`

### Wave 4 - Domain, Cache, Operator Workflows

- [ ] W4-T1: Implement domain add/remove and TLS status surfaces.
  - Depends on: MP2
  - Feature spec: `docs/features/feature-domains-and-tls.md`
- [ ] W4-T2: Implement cache toggle/purge mutations.
  - Depends on: MP2
  - Feature spec: `docs/features/feature-cache-controls.md`
- [ ] W4-T3: Implement magic login synchronous node query.
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-magic-login.md`
- [ ] W4-T4: Implement site import flow (`site_import`).
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-site-import.md`
- [ ] W4-T5: Implement settings domain config control surface.
  - Depends on: W4-T1
  - Feature spec: `docs/features/feature-settings-domain-config.md`

### Wave 5 - Observability and Control

- [ ] W5-T1: Implement jobs and metrics read APIs.
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-jobs-and-metrics.md`
- [ ] W5-T2: Implement audit logging surfaces.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-audit-logging.md`
- [ ] W5-T3: Implement backup retention cleanup orchestration.
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-backup-retention-cleanup.md`
- [ ] W5-T4: Implement administrative job control actions.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-job-control.md`

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
