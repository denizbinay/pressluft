# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-22

## Current Stage

- Stage: Wave 5 complete; Wave 5.5 runtime-first pivot queued before Wave 6.
- Blocker: none.

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

## In Progress

- Preparing Wave 5.5 implementation for self-node runnable WordPress vertical slice.

## Next Up

1. Execute W5.5-T2 to define and validate self-node local/WSL2 runtime target.
2. Execute W5.5-T3 to wire site/environment mutation execution to runnable runtime provisioning.
3. Execute W5.5-T4/W5.5-T5 to enforce reachability-gated success and add end-to-end smoke verification.

## Open Risks

- Wave 2+ work spans API, data model, and UI surfaces and may require additional scaffolding not yet present in this repository.
- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
- Active contradictory specs can force implementation outside allowed paths unless corrected before coding.
- Dashboard IA refactor can regress existing create/list behavior if subsite routing and shared context are introduced without compatibility tests.
- Local runtime networking differences (host Linux vs WSL2 DNS/ports) can break preview reachability unless self-node assumptions are explicit.

## Latest Verification Snapshot

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
