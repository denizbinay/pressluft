# ADR-0007: Introduce `ops/` Workspace for Server Operations

- Status: accepted
- Date: 2026-02-23

## Context

Server provisioning moved beyond provider credential validation and server create calls. We need a contributor-friendly area where operations specialists can define and review managed server behavior without learning backend internals.

## Decision

Create a top-level `ops/` workspace as the canonical location for operational artifacts:
- profiles
- ansible playbooks/roles
- templates
- scripts
- schemas
- tests

Existing profile references move from `infra/profiles/...` to `ops/profiles/...`.

## Consequences

### Positive
- Clear contributor boundary for ops-focused work.
- Better naming semantics than the broad `infra/` label.
- Enables long-term managed-hosting lifecycle work without coupling every change to Go.

### Trade-offs
- Requires migration updates in profile registry references.
- Adds another top-level domain that must be documented and governed.
