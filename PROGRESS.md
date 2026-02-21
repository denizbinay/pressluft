# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-21

## Current Stage

- Stage: unattended orchestration baseline established.
- Single guarded OpenCode runtime config is now canonical for all sessions.

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

- None.

## Next Up

1. Smoke-test unattended commands (`/run-plan`, `/resume-run`, `/triage-failures`).
2. Execute unattended run for Wave 0 (`/run-plan`).
3. Continue Wave 1 after Wave 0 gate stabilization.

## Open Risks

- Backend gate commands are currently conditional in CI until `cmd/pressluft/main.go` exists.
- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
