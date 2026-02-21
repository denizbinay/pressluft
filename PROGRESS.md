# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-21

## Current Stage

- Stage: Wave 11 (local sandbox) complete; packaging baseline implemented.
- Auth/session, audit logging, site/environment create foundations, backup create/list API/job wiring, health-check/rollback orchestration, deploy/update, restore, and promotion/drift guardrail mutation flows now exist alongside queue-backed node provisioning.
- Complete Nuxt dashboard scope is now accepted as MVP and scheduled for Waves 6-8.

## Completed

- Governance docs added: `docs/agent-governance.md`, `docs/parallel-execution.md`.
- Session handoff template added: `docs/templates/session-handoff-template.md`.
- Major-change proposal workflow added: `docs/changes-workflow.md`, `changes/_template/*`.
- `PLAN.md` refactored into atomic wave-based task structure with dependencies.
- Spec/contract readiness scripts added under `scripts/`.
- CI workflow added: `.github/workflows/ci.yml`.
- OpenCode agent role pack added under `.opencode/agents/`.
- OpenCode runtime bootstrap added: `opencode.json`.
- OpenCode command preset pack added under `.opencode/commands/`.
- OpenCode agent role files normalized with runnable frontmatter.
- OpenCode runtime permissions and task-delegation boundaries added to `opencode.json`.
- OpenCode quick-start runbook added to `README.md`.
- Claude compatibility shim retained as non-canonical: `CLAUDE.md`.
- Root spec router docs added: `SPEC.md`, `ARCHITECTURE.md`, `CONTRACTS.md`.
- ADR system added: `docs/adr/README.md`, `docs/adr/template.md`, `docs/adr/0001-spec-routing-and-contract-authority.md`.
- Parallel lock registry enforcement added: `coordination/locks.md`, `scripts/check-parallel-locks.sh`.
- Feature spec template and active Wave 0/1 feature specs updated with WHEN/THEN scenarios.
- Unattended orchestration docs and OpenCode command presets added (`docs/unattended-orchestration.md`, `.opencode/commands/*`).
- Unattended OpenCode command presets added (`/run-plan`, `/resume-run`, `/triage-failures`).
- Wave 0 bootstrap work completed (`W0-T1`, `W0-T2`, `W0-T3`).
- Go module and runnable control-plane entrypoint added (`go.mod`, `cmd/pressluft/main.go`).
- Bootstrap internals added for local node registration and `node_provision` enqueue idempotency (`internal/bootstrap/**`, `internal/nodes/**`, `internal/jobs/**`).
- Migration runner migrated off `sqlite3` CLI to embedded driver execution (`migrations/migrate.go`).
- CI backend gates now execute unconditionally (`.github/workflows/ci.yml`).
- Node provisioning executor added with transactional success/failure transitions and stable error truncation (`internal/jobs/node_provision_executor.go`).
- Mutation queue concurrency guardrails added for one active mutation per site and per node (`internal/jobs/repository.go`).
- Worker command path added for queue execution (`cmd/pressluft/main.go` `worker-once`).
- Auth API handlers and cookie session lifecycle added for login/logout and protected route checks (`internal/api/server.go`, `internal/auth/service.go`).
- Baseline mutating action audit writes added for auth login/logout (`internal/audit/service.go`).
- Site create APIs (`POST /api/sites`, `GET /api/sites`, `GET /api/sites/{id}`) now persist site + initial production environment, generate deterministic preview URL, and enqueue `site_create` jobs (`internal/api/server.go`, `internal/sites/service.go`).
- Environment create/list/get APIs (`POST/GET /api/sites/{id}/environments`, `GET /api/environments/{id}`) now validate staging/clone payloads, persist clone-intent state, and enqueue `env_create` jobs with site/node concurrency guards (`internal/api/server.go`, `internal/environments/service.go`).
- Backup create/list APIs (`POST/GET /api/environments/{id}/backups`) now validate backup scope, persist retention metadata, enqueue `backup_create` jobs, and expose backup lifecycle records (`internal/api/server.go`, `internal/backups/service.go`, `internal/jobs/repository.go`).
- Backup lifecycle transition tests and syntax-checked backup-create Ansible playbook baseline added (`internal/backups/service_test.go`, `ansible/playbooks/backup-create.yml`).
- Health-check and rollback orchestration foundations added for post-mutation gating (`internal/releases/service.go`) including `health_check` enqueueing, automatic `release_rollback` enqueue on failed checks, release health transitions, rollback success/failure state reconciliation, and stable structured error codes.
- Health-check and rollback orchestration tests added (`internal/releases/service_test.go`), plus baseline playbooks (`ansible/playbooks/health-check.yml`, `ansible/playbooks/release-rollback.yml`) and health-check error code documentation (`docs/health-checks.md`).
- Deploy/update mutation flows added for `POST /api/environments/{id}/deploy` and `POST /api/environments/{id}/updates` with payload validation, deploy-state transitions, release metadata creation for deploys, pre-update backup freshness checks, queued `env_deploy`/`env_update` jobs, audit writes, and lifecycle transition helpers (`internal/environments/service.go`, `internal/api/server.go`).
- Deploy/update tests and baseline playbooks added (`internal/environments/service_test.go`, `internal/api/server_test.go`, `ansible/playbooks/env-deploy.yml`, `ansible/playbooks/env-update.yml`).
- Environment restore flow added for `POST /api/environments/{id}/restore` with backup ownership/completion validation, restore-state transitions, pre-restore full backup reuse-or-create, queued `env_restore` jobs, audit writes, restore lifecycle transition helpers, tests, and baseline playbook (`internal/environments/service.go`, `internal/api/server.go`, `internal/environments/service_test.go`, `internal/api/server_test.go`, `ansible/playbooks/env-restore.yml`).
- Drift-check and promote flows added for `POST /api/environments/{id}/drift-check` and `POST /api/environments/{id}/promote` with persisted drift records, clean-drift + fresh-backup gate enforcement, queued `drift_check`/`env_promote` jobs, audit writes, promote lifecycle transition helpers, and tests (`internal/promotion/service.go`, `internal/api/server.go`, `internal/promotion/service_test.go`, `internal/api/server_test.go`).
- Promotion and drift-check baseline playbooks added (`ansible/playbooks/drift-check.yml`, `ansible/playbooks/env-promote.yml`) and API contract docs aligned (`contracts/openapi.yaml`, `docs/api-contract.md`).
- Domain add/remove/list API surfaces implemented with queued `domain_add`/`domain_remove` jobs, persisted TLS lifecycle states, deterministic DNS mismatch helper (`DOMAIN_DNS_MISMATCH`), domain service tests, API handler tests, and baseline domain playbooks (`internal/domains/service.go`, `internal/api/server.go`, `internal/domains/service_test.go`, `internal/api/server_test.go`, `ansible/playbooks/domain-add.yml`, `ansible/playbooks/domain-remove.yml`, `docs/api-contract.md`, `docs/error-codes.md`).
- Cache toggle/purge API surfaces implemented with queued `env_cache_toggle`/`cache_purge` jobs, selective cache-flag updates, payload validation for non-empty cache toggle bodies, cache service tests, API handler tests, and baseline cache playbooks (`internal/environments/service.go`, `internal/api/server.go`, `internal/environments/service_test.go`, `internal/api/server_test.go`, `ansible/playbooks/env-cache-toggle.yml`, `ansible/playbooks/cache-purge.yml`, `docs/api-contract.md`).
- Magic login synchronous node query implemented for `POST /api/environments/{id}/magic-login` with active-state validation, direct SSH execution with 10-second hard timeout, stable `environment_not_active`/`node_unreachable`/`wp_cli_error` error mapping, no job enqueueing, and audit logging (`internal/ssh/service.go`, `internal/api/server.go`, `internal/ssh/service_test.go`, `internal/api/server_test.go`, `cmd/pressluft/main.go`).
- Site import flow implemented for `POST /api/sites/{id}/import` with archive URL validation, queued `site_import` jobs, site/environment restoring transitions, import release tracking, executor retry/backoff + structured Ansible error mapping, API/import service tests, and baseline site-import playbook (`internal/migration/service.go`, `internal/migration/executor.go`, `internal/migration/service_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, `cmd/pressluft/main.go`, `ansible/playbooks/site-import.yml`).
- Settings domain config control surface implemented under authenticated internal admin API (`GET/PUT /_admin/settings/domain-config`) with deterministic full-update validation for `control_plane_domain`/`preview_domain`/`dns01_provider`/`dns01_credentials_json`, encrypted secret reference storage for DNS credentials, redacted read responses, API/service tests, and runtime secrets-dir wiring (`internal/api/server.go`, `internal/api/server_test.go`, `internal/settings/service.go`, `internal/settings/service_test.go`, `internal/secrets/store.go`, `cmd/pressluft/main.go`, `docs/domain-and-routing.md`, `docs/config-matrix.md`).
- Jobs and metrics read APIs implemented for `GET /api/jobs`, `GET /api/jobs/{id}`, and `GET /api/metrics` with stable job payloads, full job detail fields (attempt/error metadata), point-in-time DB counters, unauthorized guard coverage, and service + handler tests (`internal/jobs/service.go`, `internal/jobs/service_test.go`, `internal/metrics/service.go`, `internal/metrics/service_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, `cmd/pressluft/main.go`).
- Backup retention cleanup orchestration implemented with expired-backup scheduler enqueueing (`retention_until < now`) and site-level de-dup, `backup_cleanup` job executor retry/backoff + structured Ansible error codes, backup state transition to `expired` on cleanup success, dedicated tests for enqueue/executor behavior, backup-cleanup playbook baseline, and cleanup behavior docs alignment (`internal/backups/cleanup_scheduler.go`, `internal/backups/cleanup_executor.go`, `internal/backups/cleanup_test.go`, `ansible/playbooks/backup-cleanup.yml`, `docs/backups-and-restore.md`, `PLAN.md`).
- Administrative job control actions implemented for `POST /api/jobs/{id}/cancel`, `POST /api/sites/{id}/reset`, and `POST /api/environments/{id}/reset` with deterministic state guards (`job_not_cancellable`, `resource_not_failed`, `reset_validation_failed`), transactional service logic, API/audit wiring, and service + handler tests (`internal/jobs/service.go`, `internal/jobs/service_test.go`, `internal/sites/service.go`, `internal/sites/service_test.go`, `internal/environments/service.go`, `internal/environments/service_test.go`, `internal/api/server.go`, `internal/api/server_test.go`).
- Wave 6 dashboard bootstrap completed with a new Nuxt 3 + TypeScript workspace, strict runtime config baseline, typed API client foundation for core authenticated operations, and runnable frontend gates (`web/package.json`, `web/nuxt.config.ts`, `web/lib/api/client.ts`, `web/lib/api/types.ts`, `web/composables/useApiClient.ts`, `web/pages/index.vue`).
- Wave 6 web auth and protected shell completed with login/logout UX, deterministic 401 redirect handling, and base protected routes (`web/pages/login.vue`, `web/pages/app/index.vue`, `web/middleware/auth.global.client.ts`, `web/composables/useAuthSession.ts`).
- Wave 6 embedded dashboard delivery completed by serving built assets from the Go control-plane with SPA fallback precedence and handler tests, plus CI/frontend gate wiring (`cmd/pressluft/main.go`, `internal/api/server.go`, `internal/api/server_test.go`, `.github/workflows/ci.yml`).
- Wave 7 sites and environments dashboard flows implemented with site/environment create and deterministic post-job refresh behavior (`web/pages/app/sites/index.vue`, `web/pages/app/sites/[id].vue`, `web/pages/app/environments/[id].vue`, `web/tests/sites-and-environments.test.ts`).
- Wave 7 lifecycle workflows implemented for deploy/updates/restore/drift-check/promote with job polling and conflict/validation handling (`web/pages/app/environments/[id].vue`, `web/tests/lifecycle-workflows.test.ts`).
- Wave 7 operations workflows implemented for backups, domains, caching, and magic-login with deterministic error handling and job polling (`web/pages/app/environments/[id].vue`, `web/tests/operations-workflows.test.ts`).
- Wave 7 jobs/metrics visibility and administrative controls implemented with in-shell metrics, jobs list/detail, job cancel, and failed-state reset actions (`web/layouts/app.vue`, `web/pages/app/jobs/index.vue`, `web/pages/app/jobs/[id].vue`, `web/tests/jobs-metrics-controls.test.ts`).
- Wave 8 embedded dashboard smoke verification now uses static Nuxt output (ensures `./web/.output/public/index.html` exists) and documents correct `pressluft` command/DB-migration ordering (`web/package.json`, `web/nuxt.config.ts`, `docs/testing.md`).
- Wave 9-11 installation/packaging baseline added with curlable release-based installer, systemd units, `pressluft migrate`/`pressluft worker`/`pressluft version` commands, committed `ansible/ansible.cfg`, GitHub release workflow, and disposable local sandbox script (`install.sh`, `packaging/**`, `.github/workflows/release.yml`, `scripts/dev-sandbox.sh`, `cmd/pressluft/main.go`, `internal/migrations/**`).

## In Progress

- None.

## Next Up

- None.

## Open Risks

- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
