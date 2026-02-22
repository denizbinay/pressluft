# Pressluft MVP Plan

Status: active
Owner: platform
Last Updated: 2026-02-22

## Preconditions (Must Stay Green)

1. Spec/contract readiness checks pass: `bash scripts/check-readiness.sh`.
2. OpenAPI authority remains canonical (`contracts/openapi.yaml`).
3. Runtime bootstrap remains canonical (`opencode.json`).
4. Baseline schema applies successfully (`go run ./migrations/migrate.go up`).

## Operator Runbook (Unattended-Compatible)

1. Start unattended execution: `/run-plan`
2. Resume after interruption: `/resume-run` (optional hint like `W3-T1`)
3. Triage failures: `/triage-failures`

Progress persistence contract:

- `PLAN.md` checkboxes are the execution source of truth.
- `PROGRESS.md` tracks current stage, blockers, and next step.
- Substantial sessions write handoff notes using `docs/templates/session-handoff-template.md`.

## Wave Completion Rule (Mandatory)

A wave is complete only when all are true:

1. App starts with one command (`pressluft dev` or `pressluft serve`).
2. Dashboard is reachable in browser and reflects wave capability.
3. Terminal logs show server startup, listening port, and request handling.
4. Required verification gates pass for the changed surface.

## Wave Map

- Wave 1: runnable shell + dashboard placeholder + local dev setup.
- Wave 2: auth + jobs/metrics visibility dashboard.
- Wave 3: mutation engine + node provisioning + audit visibility.
- Wave 4: site/environment create + clone flows in dashboard.
- Wave 5: backup create/list + retention visibility.
- Wave 6: deploy/update + health checks + automatic rollback.
- Wave 7: restore + promotion with drift guardrails.
- Wave 8: domains/TLS + settings domain config control surface.
- Wave 9: operator toolkit completion (cache, magic login, import, cleanup, job control).

Merge points:

- MP1: end of Wave 3 before site/env lifecycle expansion.
- MP2: end of Wave 6 before advanced operator and routing controls.
- MP3: end of Wave 9 before MVP release hardening.

## Atomic Task Backlog

### Wave 1 - Runnable Shell and Developer Bootstrap

- [x] W1-T1: Create runnable Go app entrypoint and HTTP server (`pressluft dev`).
  - Depends on: none
  - Feature spec: `docs/features/feature-wave1-runtime-shell.md`
- [x] W1-T2: Serve dashboard placeholder page at `/` with wave status text.
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-wave1-runtime-shell.md`
- [x] W1-T3: Add local install/dev setup instructions (from zero to running).
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-wave1-runtime-shell.md`
- [x] W1-T4: Ensure startup/request logs are visible and deterministic.
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-wave1-runtime-shell.md`

Wave 1 manual test contract:

- CLI: `go run ./cmd/pressluft dev`
- Browser: `http://localhost:<port>/` shows `Wave 1 complete - features will be added incrementally`
- Logs: startup + listening port + request lines for dashboard hits

### Wave 2 - Auth and Operator Visibility First

- [x] W2-T1: Implement login/logout cookie session lifecycle.
  - Depends on: W1-T1
  - Feature spec: `docs/features/feature-auth-session.md`
- [x] W2-T2: Implement jobs/metrics read APIs for dashboard visibility.
  - Depends on: W2-T1
  - Feature spec: `docs/features/feature-jobs-and-metrics.md`
- [x] W2-T3: Update dashboard with auth screen and jobs/metrics panels.
  - Depends on: W2-T1, W2-T2
  - Feature specs:
    - `docs/features/feature-auth-session.md`
    - `docs/features/feature-jobs-and-metrics.md`

Wave 2 manual test contract:

- CLI: `pressluft dev`
- Browser: login page, then dashboard with jobs table and metrics cards
- Logs: auth attempts/results + protected API request logs

### Wave 3 - Mutation Engine, Node Provision, and Audit Trail

- [x] W3-T1: Implement transactional mutation queue worker with concurrency invariants.
  - Depends on: W2-T2
  - Feature spec: `docs/features/feature-node-provision.md`
- [x] W3-T2: Implement node provision mutation path via Ansible execution contract.
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-node-provision.md`
- [x] W3-T3: Implement baseline audit logging for mutating actions.
  - Depends on: W3-T2
  - Feature spec: `docs/features/feature-audit-logging.md`
- [x] W3-T4: Add dashboard job detail timeline for queued/running/succeeded/failed states.
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-jobs-and-metrics.md`

Wave 3 manual test contract:

- CLI: `pressluft dev`
- Browser: trigger node provision and observe live status transitions + audit-visible outcomes
- Logs: enqueue, lock acquire/release, worker execution, ansible run result, audit write

### Wave 4 - Site and Environment Creation Flows

- [ ] W4-T1: Implement site create/read/list contract and storage mapping.
  - Depends on: MP1
  - Feature spec: `docs/features/feature-site-create.md`
- [ ] W4-T2: Implement environment create/clone contract and state transitions.
  - Depends on: W4-T1
  - Feature spec: `docs/features/feature-environment-create-clone.md`
- [ ] W4-T3: Add dashboard create flows for site/environment and state display.
  - Depends on: W4-T1, W4-T2
  - Feature specs:
    - `docs/features/feature-site-create.md`
    - `docs/features/feature-environment-create-clone.md`

Wave 4 manual test contract:

- CLI: `pressluft dev`
- Browser: create site + env/clone and view resulting records/status
- Logs: request validation, job enqueue/execution, final state transitions

### Wave 5 - Backups and Recovery Foundations

- [ ] W5-T1: Implement backup create/list contract and lifecycle.
  - Depends on: W4-T2
  - Feature spec: `docs/features/feature-backups.md`
- [ ] W5-T2: Add dashboard backups surface with retention metadata.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-backups.md`

Wave 5 manual test contract:

- CLI: `pressluft dev`
- Browser: trigger backup and observe entry/state in backups UI
- Logs: backup enqueue/start/complete or failure with error code

### Wave 6 - Deploy/Update Safety with Health and Rollback

- [ ] W6-T1: Implement deploy/update mutation contracts.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-environment-deploy-updates.md`
- [ ] W6-T2: Implement health checks and rollback orchestration.
  - Depends on: W6-T1
  - Feature spec: `docs/features/feature-health-checks-and-rollback.md`
- [ ] W6-T3: Add dashboard release timeline with health outcomes and rollback events.
  - Depends on: W6-T1, W6-T2
  - Feature specs:
    - `docs/features/feature-environment-deploy-updates.md`
    - `docs/features/feature-health-checks-and-rollback.md`

Wave 6 manual test contract:

- CLI: `pressluft dev`
- Browser: run deploy/update and verify health pass/fail + rollback evidence
- Logs: deploy/update flow, health probe details, rollback trigger/completion

### Wave 7 - Restore and Promotion Guardrails

- [ ] W7-T1: Implement environment restore from backup with safety checks.
  - Depends on: MP2
  - Feature spec: `docs/features/feature-environment-restore.md`
- [ ] W7-T2: Implement drift check and guarded promotion flow.
  - Depends on: W7-T1
  - Feature spec: `docs/features/feature-promotion-drift.md`
- [ ] W7-T3: Add dashboard surfaces for blocked promotion reasons and restore outcomes.
  - Depends on: W7-T1, W7-T2
  - Feature specs:
    - `docs/features/feature-environment-restore.md`
    - `docs/features/feature-promotion-drift.md`

Wave 7 manual test contract:

- CLI: `pressluft dev`
- Browser: execute restore + promotion scenarios (pass and blocked)
- Logs: precondition checks, drift findings, restore/promotion decisions

### Wave 8 - Domain and TLS Control Surfaces

- [ ] W8-T1: Implement domain add/remove and TLS status lifecycle.
  - Depends on: W4-T2
  - Feature spec: `docs/features/feature-domains-and-tls.md`
- [ ] W8-T2: Implement internal domain settings control API/UI.
  - Depends on: W8-T1
  - Feature spec: `docs/features/feature-settings-domain-config.md`
- [ ] W8-T3: Add dashboard domain/TLS/settings screens with validation feedback.
  - Depends on: W8-T1, W8-T2
  - Feature specs:
    - `docs/features/feature-domains-and-tls.md`
    - `docs/features/feature-settings-domain-config.md`

Wave 8 manual test contract:

- CLI: `pressluft dev`
- Browser: add/remove domain, inspect TLS status, update settings
- Logs: domain mutation jobs, cert status changes, config validation events

### Wave 9 - Operator Toolkit Completion

- [ ] W9-T1: Implement cache toggle/purge mutations and UI controls.
  - Depends on: W4-T2
  - Feature spec: `docs/features/feature-cache-controls.md`
- [ ] W9-T2: Implement magic login synchronous SSH query path and UI.
  - Depends on: W4-T2
  - Feature spec: `docs/features/feature-magic-login.md`
- [ ] W9-T3: Implement site import flow and dashboard visibility.
  - Depends on: W4-T2
  - Feature spec: `docs/features/feature-site-import.md`
- [ ] W9-T4: Implement backup retention cleanup scheduler visibility.
  - Depends on: W5-T1
  - Feature spec: `docs/features/feature-backup-retention-cleanup.md`
- [ ] W9-T5: Implement job control actions (cancel/reset) with dashboard controls.
  - Depends on: W3-T1
  - Feature spec: `docs/features/feature-job-control.md`

Wave 9 manual test contract:

- CLI: `pressluft dev`
- Browser: cache actions, magic login generation, import progress, cleanup effects, cancel/reset controls
- Logs: sync SSH query logs, scheduler runs, cancel/reset decisions, import execution traces

## Dependency and Contract Rules

- `contracts/openapi.yaml` is canonical for API behavior.
- Endpoint ownership remains 1:1 in `docs/contract-traceability.md`.
- Async endpoint job types must exist in `docs/job-types.md`.
- Error surfaces must be registered in `docs/error-codes.md`.
- Enum/state values must stay aligned with `docs/data-model.md` and `docs/state-machines.md`.
- Infrastructure mutations must run through queue + Ansible.
- Database is source of truth for state transitions.

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
- Keep each wave manually runnable and browser-visible.
- No wave may end with purely internal work and no demo.
- Major changes must use `changes/<slug>/` workflow (`docs/changes-workflow.md`).
- No-forward-dependency proof is maintained in `docs/plan-dependency-matrix.md`.
