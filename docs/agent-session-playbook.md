Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, AGENTS.md, docs/templates/session-handoff-template.md, docs/agent-governance.md
Supersedes: none

# Agent Session Playbook

This guide defines minimal context packets for agent sessions.

## Session Contract

Every implementation session must:

1. Name the governing feature spec in `docs/features/`.
2. List allowed file paths for this session.
3. Follow Spec -> Plan -> Act -> Verify.
4. Report verification mapped to acceptance criteria.
5. Produce a structured handoff for substantial sessions using `docs/templates/session-handoff-template.md`.
6. Start from OpenCode instruction bootstrap in `opencode.json`.

## Operator Flow (OpenCode)

1. Run `/readiness`.
2. Run `/session-kickoff <docs/features/feature-*.md>` for scoped kickoff packet.
3. Implement with Build or delegated subagents.
4. Run `/backend-gates` and `/frontend-gates` as applicable.
5. Produce handoff output for substantial sessions.

## Unattended Flow (OpenCode)

1. Start OpenCode from repository root (or open this repo in Desktop).
2. Start unattended execution: `/run-plan`.
3. If interrupted, start a new session and resume with `/resume-run`.
4. On failure, run `/triage-failures`.

### Session Kickoff Template

Every non-trivial implementation session should start with this packet:

- Governing feature spec: `<docs/features/feature-*.md>`
- Allowed change paths: `<path globs from feature spec>`
- Explicitly forbidden paths: everything else unless scope is re-approved
- Acceptance criteria to prove: `<numbered items from feature spec>`
- Verification commands to run: `<commands from docs/testing.md and feature spec>`

OpenCode startup baseline is loaded from `opencode.json` instructions. Additional docs are loaded only as needed for the selected feature scope.

Recommended command preset for kickoff packet generation: `/session-kickoff <docs/features/feature-*.md>`.

## Minimal Context by Task Type

### Backend + DB

- `AGENTS.md`
- `docs/spec-index.md`
- one file under `docs/features/`
- `docs/technical-architecture.md`
- `docs/data-model.md`
- `docs/job-execution.md`
- `contracts/openapi.yaml` (if endpoint behavior is touched)
- `docs/contract-guardrails.md` (if contract/state changes)

### Frontend

- `AGENTS.md`
- `docs/spec-index.md`
- one file under `docs/features/`
- `docs/ui-flows.md`
- `contracts/openapi.yaml`
- `docs/contract-traceability.md`

### Infrastructure (Ansible)

- `AGENTS.md`
- `docs/spec-index.md`
- one file under `docs/features/`
- `docs/ansible-execution.md`
- `docs/provisioning-spec.md`
- `docs/security-and-secrets.md`
- `docs/contract-traceability.md` (if API-triggered jobs are affected)

## Scope Rules

- Keep each session to one feature spec.
- Avoid broad multi-layer refactors in one session.
- If scope grows, split into additional feature specs and sessions.

## Session Closeout

For substantial sessions:

- Update `PLAN.md` task checkboxes for completed work.
- Update `PROGRESS.md` stage/open risks.
- Add a handoff entry using `docs/templates/session-handoff-template.md`.
- Record blockers and decisions explicitly to prevent rework in the next session.
