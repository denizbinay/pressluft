# Pressluft MVP Progress

Status: active
Owner: platform
Last Updated: 2026-02-21

## Current Stage

- Stage: process hardening baseline complete.
- Reliability-first agent workflow and automation scaffolding established.

## Completed

- Governance docs added: `docs/agent-governance.md`, `docs/parallel-execution.md`.
- Session handoff template added: `docs/templates/session-handoff-template.md`.
- Major-change proposal workflow added: `docs/changes-workflow.md`, `changes/_template/*`.
- `PLAN.md` refactored into atomic wave-based task structure with dependencies.
- Spec/contract readiness scripts added under `scripts/`.
- CI workflow added: `.github/workflows/ci.yml`.
- OpenCode agent role pack added under `.opencode/agents/`.
- Claude compatibility shim added: `CLAUDE.md`.
- Root spec router docs added: `SPEC.md`, `ARCHITECTURE.md`, `CONTRACTS.md`.
- ADR system added: `docs/adr/README.md`, `docs/adr/template.md`, `docs/adr/0001-spec-routing-and-contract-authority.md`.
- Parallel lock registry enforcement added: `coordination/locks.md`, `scripts/check-parallel-locks.sh`.
- Feature spec template and active Wave 0/1 feature specs updated with WHEN/THEN scenarios.

## In Progress

- None.

## Next Up

1. Execute Wave 0 task W0-T1 to establish runnable Go module structure (`cmd/pressluft`, `internal/**`).
2. Execute W0-T2 and W0-T3 to make backend gates fully runnable in CI.
3. Start Wave 1 with node provision and auth-session foundations.

## Open Risks

- Backend gate commands are currently conditional in CI until `cmd/pressluft/main.go` exists.
- Parallel execution now has lock linting, but still depends on active discipline for ownership transfer during live parallel sessions.
