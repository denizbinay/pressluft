# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-22

## Current Stage

- Stage: Wave 1 runtime shell completed; ready to begin Wave 2 auth and visibility tasks.
- Blocker: none for Wave 1.

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

## In Progress

- Planning W2-T1 (`docs/features/feature-auth-session.md`) implementation scope.

## Next Up

1. Execute W2-T1 login/logout cookie session lifecycle from `docs/features/feature-auth-session.md`.
2. Implement W2-T2 jobs/metrics read APIs and wire into dashboard view model.
3. Implement W2-T3 dashboard auth screen and jobs/metrics panels with acceptance verification.

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

## Session Handoff (2026-02-22)

### 1) State Snapshot

- Session date/time: 2026-02-22 UTC.
- Branch/worktree: current branch in `/home/deniz/projects/pressluft`.
- Governing feature spec: `docs/features/feature-wave1-runtime-shell.md`.
- Tasks completed: W1-T1, W1-T2, W1-T3, W1-T4.
- Tasks in progress: W2-T1 planning.
- Tasks blocked: none.
- Verification summary (pass/fail): pass for readiness, build, vet, tests, and browser smoke path.

### 2) Narrative Context

- Why this session focused on the selected scope: unattended execution was blocked because Wave 1 paths had no owning feature spec.
- Key implementation intent: create a minimal runnable shell that is browser-visible and deterministic for logs/testing.
- Important non-obvious tradeoffs: kept response as plain text and avoided early auth/data scaffolding to preserve Wave 1 minimality.

### 3) Decisions Made

- Decision: introduced a dedicated Wave 1 feature spec and repointed Wave 1 tasks to it.
  - Rationale: resolve allowed-path ownership mismatch without broadening install bootstrap scope.
  - Spec/contract impact: `PLAN.md` now maps W1 tasks to `docs/features/feature-wave1-runtime-shell.md`; no API/schema contract changes.
- Decision: implemented `dev` and `serve` through the same HTTP runtime path.
  - Rationale: reduce early command divergence while meeting Wave 1 run requirements.
  - Spec/contract impact: no OpenAPI or DB impact.

### 4) Priorities and Next Steps

1. First next task: execute W2-T1 login/logout cookie session lifecycle.
2. Second next task: implement W2-T2 jobs/metrics read APIs.
3. Verification required before moving phases: rerun backend gates after each W2 milestone and preserve browser-visible behavior.

### 5) Warnings and Blockers

- Known risk: Wave 2+ may require additional scaffolding for persistence and handlers.
- Open blocker: none currently.
- Avoid this pitfall next session: do not start implementation for any wave task without a feature spec that explicitly owns intended paths.
