# Proposal: wp-first-runtime

## Problem

Pressluft has strong control-plane progress across jobs, API, and dashboard flows, but operator value is blocked when create-site does not reliably yield a reachable WordPress runtime URL. We need a focused vertical slice that makes runtime reachability the first-class acceptance gate before deeper safety automation work.

## Scope

- In scope:
  - Introduce a WordPress-first runtime milestone before Wave 6.
  - Define self-node behavior for local/WSL2 execution.
  - Wire site/environment mutation execution to provision runnable runtime artifacts.
  - Require runtime reachability for mutation success semantics.
  - Add deterministic smoke verification for create-site to preview URL loadability.
- Out of scope:
  - New public API endpoints.
  - Broad redesign of post-Wave-6 operator features.
  - Multi-region or multi-node balancing strategy.

## Impact Summary

- API contract: none
- DB schema/migrations: none
- Infra/playbooks: update-required
- Security posture: none

## Governing Specs

- `docs/spec-index.md`
- `AGENTS.md`
- `docs/agent-governance.md`
- `docs/features/feature-wp-first-runtime.md`
- `docs/ansible-execution.md`
- `docs/provisioning-spec.md`
- `docs/domain-and-routing.md`

## Acceptance Criteria

1. Plan order is corrected so Wave 6+ depends on a completed runtime vertical slice milestone (MP1.5).
2. Site creation on self-node baseline yields reachable WordPress preview URL as an explicit manual and scripted acceptance gate.
3. Job success semantics align with runtime reachability, preventing false-positive completion when runtime is not usable.
