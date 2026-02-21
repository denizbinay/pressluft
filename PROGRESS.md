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

## In Progress

- None.

## Next Up

1. Execute Wave 0 task W0-T1 to establish runnable Go module structure (`cmd/pressluft`, `internal/**`).
2. Execute W0-T2 and W0-T3 to make backend gates fully runnable in CI.
3. Start Wave 1 with node provision and auth-session foundations.

## Open Risks

- Backend gate commands are currently conditional in CI until `cmd/pressluft/main.go` exists.
- Parallel execution policy is documented but requires active lock-tracking discipline during implementation.
