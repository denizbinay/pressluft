# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-22

## Current Stage

- Stage: Wave 2 complete (W2-T1, W2-T2, W2-T3); ready to begin Wave 3 mutation engine work (W3-T1).
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

## In Progress

- Planning W3-T1 transactional mutation queue worker scope.

## Next Up

1. Implement W3-T1 transactional mutation queue worker with concurrency invariants.
2. Implement W3-T2 node provision mutation path via Ansible execution contract.
3. Implement W3-T3 baseline audit logging for mutating actions.

## Open Risks

- Wave 2+ work spans API, data model, and UI surfaces and may require additional scaffolding not yet present in this repository.
- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
- Active contradictory specs can force implementation outside allowed paths unless corrected before coding.

## Latest Verification Snapshot

- `bash scripts/check-readiness.sh`: pass.
- `go build -o ./bin/pressluft ./cmd/pressluft`: pass.
- `go vet ./...`: pass.
- `go test ./internal/... -v`: pass.
- `go run ./cmd/pressluft dev --port 18080` + `curl http://127.0.0.1:18080/`: pass (HTTP 200 + expected Wave 1 placeholder text and request log line).
- `go test ./internal/... -v`: pass (includes auth login/logout, jobs/metrics API coverage, and metrics aggregation tests).
- `go run ./cmd/pressluft dev --port 18080` + login/metrics/jobs curl smoke: pass (`/` 200, `/api/jobs` 200 with session cookie, `/api/metrics` returned non-negative counters).
- `make build`, `make vet`, `make test`: pass.
- `make dev PORT=18080` + `curl http://127.0.0.1:18080/`: pass (HTTP 200 dashboard response + request log line).

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
