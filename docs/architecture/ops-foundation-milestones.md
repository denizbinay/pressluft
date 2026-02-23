# Ops Foundation Milestones

## Objective

Establish architecture and repository foundations for:
- ops workspace contributions
- orchestration lifecycle persistence
- runner safety boundaries
- event-stream UX scaffolding

## Milestones

1. Workspace migration to `ops/` and profile schema contracts
2. Job lifecycle persistence model (`jobs`, `job_steps`, `job_events`, `job_checkpoints`)
3. Orchestrator API skeleton and event streaming path
4. Runner adapter guardrails for safe Ansible execution
5. Dashboard job timeline and retry-facing UX hooks

## Non-goals (this phase)

- Full autonomous host maintenance
- Full drift remediation loop
- Provider-specific deep execution optimizations

## Acceptance Snapshot

- `ops/` exists and is discoverable for contributors
- profiles have a schema contract
- orchestration state model compiles and persists
- job events are queryable/streamable
- trailing-zero price formatting removed from server size labels
