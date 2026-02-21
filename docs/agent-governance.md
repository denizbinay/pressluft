Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: AGENTS.md, docs/spec-index.md, docs/agent-session-playbook.md, docs/parallel-execution.md
Supersedes: none

# Agent Governance

This document defines mandatory behavioral boundaries for agentic execution.

## Always

- Follow `Spec -> Plan -> Act -> Verify` for every non-trivial task.
- Start from `docs/spec-index.md` and load only the minimal relevant context.
- Execute only within allowed paths declared by the governing feature spec.
- Keep `PLAN.md` and `PROGRESS.md` current during substantial sessions.
- Map verification output to explicit acceptance criteria.
- Keep OpenAPI-first authority for all API behavior changes.

## Ask First

- Adding Go or npm dependencies.
- Changing DB schema or files under `migrations/`.
- Adding new API endpoints.
- Changing Ansible playbook structure.
- Running destructive operations with irreversible impact.

## Never

- Bypass the job queue for infrastructure mutations.
- Introduce CGo.
- Invent endpoint/schema behavior outside specs.
- Commit secrets, keys, or `.env` files.
- Run broad refactors without an approved spec.

## Escalation Policy

### Two-Strikes-and-Pivot

If a task fails twice with materially similar approaches:

1. Stop retrying the same strategy.
2. Record attempted approaches and failure signals.
3. Escalate to planning mode and propose 1-2 alternatives.
4. Resume execution only with a changed plan.

### Blocker Escalation

Escalate immediately when blocked by:

- Missing acceptance criteria.
- Contradictory active specs.
- Unknown ownership for contract or state surface.

## Parallel Safety Rules

- One agent may own one file at a time.
- Ownership must be explicit before edits (lock record or task assignment).
- Active ownership records are tracked in `coordination/locks.md`.
- Stale ownership older than 2 hours may be reclaimed with a log entry.
- Merge points must be declared before parallel branches converge.

## Session Deliverables

Substantial sessions must output:

- Governing specs used.
- Planned touched paths.
- What changed and why.
- Verification commands and pass/fail summary.
