# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-22

## Current Stage

- Stage: Wave 5-only hardening execution; Wave 6+ remains paused until Wave 5.11 SDK-backed provider acquisition closeout is complete.
- Blocker: Wave 5.11 closeout still pending provider-backed smoke evidence (`W5.11-T9`, `W5.11-T10`).

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
- W2-T1 completed: login/logout cookie session lifecycle with session creation, expiry/revocation checks, and auth API tests.
- Repo-local developer command workflow added with `Makefile` (`make dev`, `make build`, `make vet`, `make test`, `make backend-gates`) and documented in `README.md`.
- W3-T1 completed: transactional in-memory mutation queue worker with atomic claim semantics, site/node concurrency guards, retry backoff scheduling (1m/5m/15m), and job error truncation.
- W3-T2 completed: node provisioning mutation path wired to Ansible execution contract with dynamic inventory/extra-vars generation, exit-code classification, node status transitions, and syntax-checked `ansible/playbooks/node-provision.yml`.
- W3-T3 completed: baseline audit logging added for mutating auth actions and async job lifecycle semantics with acceptance-time/result-update coverage, including audit failure-path tests.
- W3-T4 completed: dashboard now includes job detail timeline panel that renders queued/running/succeeded/failed progression from `/api/jobs/{id}` and keeps selected job detail fresh on refresh.
- W4-T1 completed: implemented `POST /api/sites`, `GET /api/sites`, and `GET /api/sites/{id}` with in-memory site/environment persistence, deterministic production preview URL generation, `site_create` job enqueueing, and enqueue-time site/node mutation conflict guards.
- W4-T2 completed: implemented `POST /api/sites/{id}/environments`, `GET /api/sites/{id}/environments`, and `GET /api/environments/{id}` with in-memory environment persistence, non-production clone/staging validation, site/environment cloning-state intent updates, and `env_create` job enqueueing.
- W4-T3 governance unblock completed: added `docs/features/feature-wave4-dashboard-create-flows.md` with explicit allowed paths and acceptance criteria for dashboard site/environment create flows.
- W4-T3 completed: dashboard now supports site and environment create forms, renders site/environment state tables, and shows inline contract-aligned `400`/`404`/`409` feedback for create-flow failures.
- W5-T1 completed: implemented `POST /api/environments/{id}/backups` and `GET /api/environments/{id}/backups` with `backup_scope` validation, `backup_create` enqueueing, in-memory backup lifecycle records (`pending|running|completed|failed|expired`) including retention metadata, backup lifecycle handler tests, and syntax-checked `ansible/playbooks/backup-create.yml`.
- W5-T2 completed: dashboard now includes backup create controls, environment-scoped backup listing, and retention metadata visibility with inline contract-aligned `400`/`404`/`409` feedback.
- W5-T3 completed: finalized `docs/features/feature-dashboard-ia-overhaul.md` and completed the major-change packet under `changes/dashboard-ia-overhaul/` (`proposal.md`, `design.md`, `tasks.md`) with Wave 5 task mapping and acceptance coverage.
- W5-T4 completed: dashboard now exposes route-level subsites and shell navigation for `/`, `/sites`, `/environments`, `/backups`, and `/jobs`, with route-aware section toggling and deep-link shell serving.
- W5-T5 completed: refactored embedded dashboard script into concern-scoped controllers/modules (shell/auth/overview/sites/environments/backups/jobs/data), centralized shared state maps/selectors, and preserved contract-aligned create/list/timeline behavior across subsite routes.
- W5-T6 completed: migrated dashboard flows fully into subsite hierarchy by wiring shared site context controls into environments/backups routes, adding explicit backup site selection, and preserving contract-aligned create/list/timeline behavior without cross-route coupling.
- W5-T7 completed: expanded dashboard regression tests with concern-scoped marker assertions across overview/sites/environments/backups/jobs and maintained subsite route/404 coverage.
- Course correction packet drafted for runtime-first pivot before Wave 6: `changes/wp-first-runtime/` and `docs/features/feature-wp-first-runtime.md`.
- W5.5-T1 completed: major-change packet and feature spec authored for runtime-first pivot before Wave 6 (`changes/wp-first-runtime/*`, `docs/features/feature-wp-first-runtime.md`).
- W5.5-T2 completed: defined and validated self-node runtime target for local/WSL2 execution with `IsLocal` node flag, `SelfNodeID` constant, `EnsureSelfNode` store method, and site service integration to use self-node as default target.
- W5.5-T3 completed: wired `site_create` and `env_create` execution handlers with local-node inventory generation, structured Ansible error mapping, and handler coverage for success and failure semantics.
- W5.5-T4 completed: added `site-create.yml` and `env-create.yml` provisioning playbooks with runtime reachability probes so job success remains gated on WordPress URL availability.
- Wave 5.6 planning kickoff completed: added `changes/dashboard-runtime-realignment/` packet (`proposal.md`, `design.md`, `tasks.md`) and feature specs for site-centric dashboard hierarchy plus runtime inventory queries (`docs/features/feature-dashboard-site-centric-hierarchy.md`, `docs/features/feature-runtime-inventory-queries.md`).
- W5.6-T1 completed: authored nodes-first/site-centric dashboard realignment planning artifacts and feature specs to stage implementation before Wave 6.
- W5.5-T5 completed: wired queue worker execution loop into `pressluft dev` runtime path with `site_create` and `env_create` handlers, 2-second polling interval, and structured logging for worker start/stop/job-processed events.
- W5.6-T2 completed: implemented `GET /api/nodes` and `GET /api/environments/{id}/wordpress-version` with nodes list API, synchronous SSH-based WordPress version query with 10-second timeout, stable error codes (`environment_not_active`, `node_unreachable`, `wp_cli_error`), audit logging for query invocations, and full test coverage for success/failure paths.
- W5.6-T3 completed: dashboard route hierarchy now serves `/`, `/nodes`, `/sites`, and `/jobs`, while dedicated top-level `/environments` and `/backups` routes return `404`.
- W5.6-T4 completed: environment and backup create/list workflows are now site-scoped panels under `/sites` only.
- W5.6-T5 completed: removed dev-dashboard seed records for jobs/sites/nodes and switched to runtime-truth empty states (with self-node ensured by runtime bootstrap).
- W5.6-T6 completed: added nodes truth panel (local-node readiness + deploy-ready signal) and expanded sites panel columns for node placement, status, preview URL, and live WordPress version query outcomes.
- W5.7-T1 completed: finalized feature spec for dedicated site details route and row quick actions (`docs/features/feature-site-detail-drilldown.md`) and inserted Wave 5.7 tasks into `PLAN.md` before Wave 6.
- W5.7-T2 completed: split dashboard routing between `/sites` index and `/sites/{site_id}` detail, moved environment/backup management to detail-only sections, and preserved top-level `/environments` and `/backups` as `404` routes.
- W5.7-T3 completed: added per-site three-dot quick actions menu (open details/create environment/create backup) with deep-link handling for `/sites/{site_id}?focus=environment|backup` and updated `internal/devserver/server_test.go` coverage.
- Wave 5-only extension planning completed: added Wave 5.9 and Wave 5.10 backlog to `PLAN.md` and authored feature specs for runtime readiness, runtime e2e smokes, and backup/restore vertical slice hardening.
- W5.8-T1 completed: captured Wave 5 stabilization bug list and acceptance addenda in feature specs for site detail drilldown and dashboard site-centric hierarchy.
- W5.8-T2 completed: fixed Wave 5 dashboard stabilization issues by constraining valid site-detail shell routes, adding deterministic local-node host fallback on `/nodes`, and failing fast on missing-site create attempts in `/sites/{site_id}` environment/backup forms.
- W5.8-T3 completed: refreshed `internal/devserver/server_test.go` route regression coverage for invalid site-detail paths and reran manual `/nodes` + `/sites` + `/sites/{site_id}` route smoke checks.
- W5.9-T1 completed: added node runtime readiness model with stable reason codes, guidance mapping, probe integration, and `GET /api/nodes` readiness payload expansion.
- W5.9-T2 completed: enforced readiness preflight gates for `POST /api/sites` and `POST /api/sites/{id}/environments` with fast-fail `409 node_not_ready` responses before enqueueing jobs.
- W5.9-T3 completed: updated dashboard `/nodes` readiness UX with per-node reason codes/guidance and propagated actionable preflight failure messaging in create flows.
- Wave 5 runtime/e2e scripts expanded: added deterministic diagnostics and new scripts for clone preview and backup/restore smokes (`scripts/smoke-site-clone-preview.sh`, `scripts/smoke-backup-restore.sh`) with explicit `409 node_not_ready` capture.
- Wave 5 backup/restore vertical-slice implementation added: real backup artifact generation/checksum+size metadata, `env_restore` API/service/job handler wiring, `ansible/playbooks/env-restore.yml`, and site/environment restore status transitions with regression tests.
- Wave 5.11 extension planning drafted: added `docs/features/feature-wave5-node-acquisition-parity.md`, `changes/node-acquisition-parity/*`, and updated Wave 5.11 backlog in `PLAN.md` with explicit `Create Local Node`/`Create Remote Node` no-fallback semantics.
- W5.10-T1 completed: backup execution now writes real artifacts with checksum/size metadata integrity checks and regression coverage.
- W5.10-T2 completed: `POST /api/environments/{id}/restore` + `env_restore` service/handler/playbook path implemented with pre-restore backup guard semantics.
- W5.10-T3 completed: `/sites/{site_id}` detail now includes restore flow controls with confirmation and terminal outcome messaging.
- W5.11-T1 completed: node acquisition parity major-change packet and feature specs are finalized under `changes/node-acquisition-parity/` and `docs/features/feature-wave5-node-acquisition-parity.md`.
- W5.11-T2 completed: added `POST /api/nodes` with explicit `acquisition_source` (`local|remote`) and strict no-fallback validation, plus `/nodes` UI controls exposing exactly `Create Local Node` and `Create Remote Node` actions with deterministic success/error messaging.
- W5.11-T4 completed: remote node creation contract scaffolding is in place without changing downstream provisioning/readiness semantics.
- W5.11 provider-first reset applied: prior local/manual parity artifacts remain historical only and are superseded by provider-first Wave 5.11 tasks in `PLAN.md`.
- Wave 5.11 planning alignment completed: plan/spec/testing contracts now require `multipass`-backed local acquisition, no success-path self-node autoseed dependency, and acquired-node smoke evidence before Wave 5 closeout.
- Wave 5.11 provider-parity planning alignment completed: local acquisition is now specified as provider-equivalent lifecycle (`create/start VM -> inject Pressluft-managed SSH key -> provision`) with explicit prohibition on provider-internal key-path dependency.
- Wave 5.11 course-correction completed: provider-first Wave 5.11 plan/tasks now replace local/manual node acquisition scope, with new feature specs for provider connections and Hetzner-backed acquisition.
- W5.11-T2 completed: provider connection control plane and `/providers` dashboard route are implemented with persisted provider secret handling and masked API responses.
- W5.11-T3 completed: `/api/nodes` is now provider-backed (`provider_id`, `name?`) and no longer accepts `acquisition_source=local|remote` or manual SSH input fields.
- W5.11-T4 completed: `node_provision` now executes Hetzner async acquisition lifecycle (`create server -> poll action -> fetch server`) before provisioning, with deterministic provider error mapping and handler/acquirer test coverage.
- W5.11-T5 completed: cleanup pass removed active local-acquisition execution branch from node provisioning, switched site-create target selection to provider-backed nodes, updated dashboard readiness messaging, and replaced Wave 5 smoke scripts with provider-token driven acquisition flow.
- W5.11-T6 completed: provider connect now validates bearer credentials via live Hetzner API checks (no `hcloud_` prefix heuristic), persists deterministic `connected|degraded` health status, and adds regression coverage for live-validation outcomes.
- Wave 5.11 replan completed: backlog expanded to `W5.11-T6` through `W5.11-T10` for bearer-token validation, `hcloud-go` migration, dashboard/API guidance alignment, and mandatory smoke-evidence closeout.
- W5.11-T7 completed: node acquisition now uses `hcloud-go` (`SSH key get/create -> server create -> action poll -> server fetch`) with deterministic `PROVIDER_*` classification preserved via typed Hetzner API status errors and updated acquirer regression coverage.
- W5.11-T8 completed: dashboard and API guidance now reference bearer-token credentials for `/providers` and `/nodes`, removing prefix-oriented UX copy while preserving existing request/response contracts.

## In Progress

- W5.11-T9 implementation kickoff: extend Wave 5 smoke/regression coverage for SDK-backed provider diagnostics.

## Next Up

1. Execute `W5.11-T9`: extend smoke/regression coverage for SDK-backed provider diagnostics.
2. Execute `W5.11-T10`: run clone + backup/restore smokes and capture provider acquisition/provision/readiness evidence.
3. Capture provider-backed Wave 5 closeout artifacts in handoff notes for MP1.5 readiness.

## Open Risks

- Wave 2+ work spans API, data model, and UI surfaces and may require additional scaffolding not yet present in this repository.
- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
- Active contradictory specs can force implementation outside allowed paths unless corrected before coding.
- Dashboard IA refactor can regress existing create/list behavior if subsite routing and shared context are introduced without compatibility tests.
- Provider API availability/rate limits can delay acquisition convergence and make smoke runtimes flaky.
- Credential persistence and masking mistakes in provider connection flow can create security exposure if not validated.
- Hetzner API/schema drift can break adapter assumptions without robust error mapping and retries.
- Live WordPress version queries can add latency to `/sites` rendering unless timeout and fallback behavior are explicit.

## Latest Verification Snapshot

- `bash scripts/check-readiness.sh`: pass.
- `GOTOOLCHAIN=local /usr/local/go/bin/go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `GOTOOLCHAIN=local /usr/local/go/bin/go vet ./...`: pass.
- `GOTOOLCHAIN=local /usr/local/go/bin/go test ./internal/... -v`: pass (includes `internal/providers/hetzner` SDK-backed acquisition tests and provider API error classification path).
- `bash scripts/check-readiness.sh`: pass.
- `/usr/local/go/bin/go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `/usr/local/go/bin/go vet ./...`: pass.
- `/usr/local/go/bin/go test ./internal/... -v`: pass (includes new provider credential live-validation coverage and provider connect status transitions).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes `POST /api/nodes` contract tests and `/nodes` UI marker coverage for explicit local/remote create actions).
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/bootstrap` acquisition adapter tests and `/api/nodes` local-acquired-target upsert coverage).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/backup-create.yml`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/env-restore.yml`: pass.
- `bash scripts/check-local-runtime-prereqs.sh`: fail (`passwordless sudo` and `wp` missing on local host runtime path).
- `bash scripts/smoke-create-site-preview.sh`: fail (`409 node_not_ready` with `sudo_unavailable, runtime_missing`).
- `bash scripts/smoke-site-clone-preview.sh`: fail (`409 node_not_ready` with `sudo_unavailable, runtime_missing`).
- `bash scripts/smoke-backup-restore.sh`: fail (`409 node_not_ready` with `sudo_unavailable, runtime_missing`).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new readiness model tests, API preflight `node_not_ready` coverage, and dashboard route regressions).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/devserver/server_test.go` coverage for invalid `/sites/` and nested `/sites/{id}/...` route rejection).
- `go run ./cmd/pressluft dev --port 18620` + `curl http://127.0.0.1:18620/sites` + `curl http://127.0.0.1:18620/sites/test-site` + `curl http://127.0.0.1:18620/sites/` + `curl http://127.0.0.1:18620/sites/test-site/nested` + `curl http://127.0.0.1:18620/nodes`: pass (`200` for `/sites`, `/sites/test-site`, `/nodes`; `404` for `/sites/` and `/sites/test-site/nested`).
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes updated `internal/devserver/server_test.go` coverage for `/sites/{site_id}` route serving and site quick-action markers).
- `go run ./cmd/pressluft dev --port 18600` + `curl http://127.0.0.1:18600/sites` + `curl http://127.0.0.1:18600/sites/test-site` + `curl http://127.0.0.1:18600/environments` + `curl http://127.0.0.1:18600/backups`: pass (`200` for `/sites` and `/sites/test-site`; `404` for `/environments` and `/backups`).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new concern-marker assertions in `internal/devserver/server_test.go`).
- `./bin/pressluft dev --port 18500` + site create API call: pass (worker loop starts with `event=worker_start`, picks up `site_create` job, and logs `event=worker_job_processed`).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes updated `internal/devserver/server_test.go` route/marker coverage for `/nodes` and removed top-level `/environments`/`/backups`).
- `go run ./cmd/pressluft dev --port 18400` + `curl http://127.0.0.1:18400/{,nodes,sites,jobs}` + `curl http://127.0.0.1:18400/{environments,backups}`: pass (`200` for `/`, `/nodes`, `/sites`, `/jobs`; `404` for `/environments`, `/backups`).
- `bash scripts/check-readiness.sh`: pass (post-plan-extension for Wave 5.6 docs/change packet).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/sites/handler_test.go` and `internal/environments/handler_test.go` coverage for runtime provisioning handlers).
- `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/site-create.yml`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/env-create.yml`: pass.
- `bash scripts/smoke-create-site-preview.sh`: fail (`expected job to succeed, got status=queued`; dev runtime has no active worker loop).
- `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/site-create.yml`: pass.
- `ansible-playbook --syntax-check ansible/playbooks/env-create.yml`: pass.
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new concern-marker assertions in `internal/devserver/server_test.go`).
- `go run ./cmd/pressluft dev --port 18310` + `curl http://127.0.0.1:18310/{,sites,environments,backups,jobs}`: pass (startup + request logs show all Wave 5 subsite routes returning `200`).
- `bash scripts/check-readiness.sh`: pass (resume-run baseline before continuing Wave 5).
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/devserver/server_test.go` coverage for subsite routes and unknown-route `404`).
- `go run ./cmd/pressluft dev --port 18180` + `curl http://127.0.0.1:18180/{,sites,environments,backups,jobs}`: pass (all subsite routes returned dashboard shell HTML).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes `internal/devserver` coverage after concern-boundary refactor).
- `go run ./cmd/pressluft dev --port 18291` + `curl http://127.0.0.1:18291/{,sites,environments,backups,jobs}`: pass (startup + request logs show all dashboard subsite routes returning `200`).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new site API handler validation tests, site persistence + `site_create` enqueue tests, and enqueue-time site/node conflict guard coverage).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass.
- `go run ./cmd/pressluft dev --port 18080` + `curl http://127.0.0.1:18080/`: pass (HTTP 200 + expected Wave 1 placeholder text and request log line).
- `go test ./internal/... -v`: pass (includes auth login/logout, jobs/metrics API coverage, and metrics aggregation tests).
- `go run ./cmd/pressluft dev --port 18080` + login/metrics/jobs curl smoke: pass (`/` 200, `/api/jobs` 200 with session cookie, `/api/metrics` returned non-negative counters).
- `make build`, `make vet`, `make test`: pass.
- `make dev PORT=18080` + `curl http://127.0.0.1:18080/`: pass (HTTP 200 dashboard response + request log line).
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/jobs` queue worker + retry/concurrency tests).
- `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new `internal/audit` tests, auth audit write/failure-path coverage, and async job audit lifecycle test).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes updated dashboard shell test coverage for job timeline panel).
- `go run ./cmd/pressluft dev --port 18123` + `curl http://127.0.0.1:18123/`: pass (HTML contains `Job Timeline` panel and request served).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new environment create/list/get handler coverage, environment service state/concurrency tests, and contract enum updates for environment creation type validation).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes updated `internal/devserver/server_test.go` assertions for site/environment dashboard create-flow elements).
- `go run ./cmd/pressluft dev --port 18140` + `curl http://127.0.0.1:18140/`: pass (HTML contains `site-form`, `environment-form`, `sites-body`, and `environments-body` markers).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new backup API handler coverage, backup service enqueue/list tests, backup lifecycle handler/store transition tests).
- `ansible-playbook --syntax-check ansible/playbooks/backup-create.yml`: pass.
- `go run ./cmd/pressluft dev --port 18155` + `curl http://127.0.0.1:18155/`: pass (HTML contains `backup-form`, `backup-environment`, and `backups-body` markers).
- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass (includes new self-node store tests, site service self-node target tests, and updated API router tests with node store injection).
- `./bin/pressluft dev --port 18340` + `curl http://127.0.0.1:18340/`: pass (startup + request logs show dashboard response).

## Session Handoff (2026-02-22)

### 1) State Snapshot

- Session date/time: 2026-02-22 UTC.
- Branch/worktree: current branch in `/home/deniz/projects/pressluft`.
- Governing feature specs: `docs/features/feature-auth-session.md`, `docs/features/feature-jobs-and-metrics.md`, `docs/features/feature-wave1-runtime-shell.md`.
- Tasks completed: W2-T1, W2-T2, W2-T3; repo-local developer command workflow (`make dev`, `make build`, `make vet`, `make test`, `make backend-gates`).
- Tasks in progress: W3-T1 planning.
- Tasks blocked: none.
- Verification summary (pass/fail): pass for readiness, backend gates, auth/jobs/metrics tests, and browser/API smoke paths.

### 2) Narrative Context

- Why this session focused on the selected scope: complete Wave 2 operator-visible baseline and remove friction in local startup commands for follow-on implementation sessions.
- Key implementation intent: ship working auth + jobs/metrics visibility APIs and dashboard, then standardize repo-local developer commands for clone-and-run workflows.
- Important non-obvious tradeoffs: keep in-memory placeholder job/metrics data for Wave 2 visibility while preserving spec alignment for upcoming Wave 3 queue/mutation work.

### 3) Decisions Made

- Decision: complete Wave 2 by delivering login/logout session lifecycle, jobs/metrics read APIs, and dashboard integration.
  - Rationale: unlock operator visibility baseline before mutation/queue implementation starts.
  - Spec/contract impact: aligned implementation with existing OpenAPI paths for `/api/login`, `/api/logout`, `/api/jobs`, `/api/jobs/{id}`, and `/api/metrics`; no DB schema change.
- Decision: adopt repo-local `Makefile` commands as recommended day-to-day workflow.
  - Rationale: avoid global PATH dependency and provide deterministic clone-and-run commands for all agents/operators.
  - Spec/contract impact: updated `README.md`, `docs/testing.md`, and `docs/features/feature-wave1-runtime-shell.md`; no API/schema/infra contract change.

### 4) Priorities and Next Steps

1. First next task: execute W3-T1 transactional mutation queue worker with concurrency invariants under `docs/features/feature-node-provision.md`.
2. Second next task: execute W3-T2 node provision mutation path via Ansible execution contract.
3. Verification required before moving phases: run `make backend-gates` and Wave 3 manual flow smoke checks (`pressluft dev`, node-provision enqueue/state transitions, lock acquire/release logs).

### 5) Warnings and Blockers

- Known risk: Wave 3 concurrency invariants (`max 1 mutation job per site` and `max 1 per node`) can regress if worker locking is implemented outside DB transaction boundaries.
- Open blocker: none currently.
- Avoid this pitfall next session: do not bypass job queue or Ansible execution contract while implementing node mutation paths.
