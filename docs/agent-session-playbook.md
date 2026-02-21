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

### Session Kickoff Template

Every non-trivial implementation session should start with this packet:

- Governing feature spec: `<docs/features/feature-*.md>`
- Allowed change paths: `<path globs from feature spec>`
- Explicitly forbidden paths: everything else unless scope is re-approved
- Acceptance criteria to prove: `<numbered items from feature spec>`
- Verification commands to run: `<commands from docs/testing.md and feature spec>`

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
