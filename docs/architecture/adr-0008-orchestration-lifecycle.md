# ADR-0008: Separate Convergence from Orchestration Lifecycle

- Status: accepted
- Date: 2026-02-23

## Context

Configuration convergence alone is not enough for product UX. Managed provisioning needs durable state, retries, resumability, and live dashboard feedback.

## Decision

Use a split model:
- Convergence layer: Ansible artifacts under `ops/ansible/`
- Orchestration layer: Go services under `internal/orchestrator`, `internal/runner`, and `internal/events`

Initial state machine:

`queued -> preparing -> running -> waiting_reboot -> resuming -> verifying -> succeeded`

Error/cancel branches:
- `running|resuming -> retrying -> running`
- `any active -> failed|cancelled|timed_out`

## Consequences

### Positive
- Product-grade lifecycle handling and dashboard visibility.
- Clear ownership boundaries across web/backend/ops contributors.

### Trade-offs
- More code than directly invoking Ansible in handlers.
- Requires explicit contracts and migration-backed state persistence.
