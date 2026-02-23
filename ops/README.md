# Ops Workspace

This directory is the operational workspace for Pressluft-managed servers.

Goals:
- Give infra/ops contributors a clear place to contribute without touching Go internals.
- Keep executable server behavior auditable in repository-native artifacts.
- Separate configuration convergence logic from backend orchestration logic.

Directory layout:
- `profiles/` declarative profile contracts and policy
- `ansible/` convergence logic (playbooks and roles)
- `templates/` reusable config templates consumed by automation
- `scripts/` thin helper scripts (bootstrap/diagnostics)
- `schemas/` profile schema definitions and validation contracts
- `tests/` ops validation guidance and test harness notes

Execution model:
- Go orchestration in `internal/` handles lifecycle state, retries, and events.
- Ops logic in this directory handles desired server configuration outcomes.
- Runtime execution must follow Ansible guardrails documented in context and ADRs.
