Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, AGENTS.md, contracts/openapi.yaml
Supersedes: none

# ADR 0001: Spec Routing and Contract Authority

## Status

accepted

## Context

The repository has a broad spec surface across architecture, data model, contracts, and workflow docs. Agent sessions need a deterministic entry path and clear authority precedence to avoid conflicting implementations and contract drift.

## Decision

- Establish top-level router docs: `SPEC.md`, `ARCHITECTURE.md`, and `CONTRACTS.md`.
- Keep `docs/spec-index.md` as the canonical document index for full navigation.
- Keep `contracts/openapi.yaml` as the authoritative API contract source.
- Require contract freshness updates in the same change when behavior changes.

## Consequences

- Positive: agents and humans get predictable entry points with minimal ambiguity.
- Positive: contract-first behavior remains explicit and enforceable.
- Negative: one more documentation layer must be kept current.
- Neutral: canonical technical detail remains in `docs/` and `contracts/`.

## Alternatives Considered

1. Keep only deep docs under `docs/` without top-level routers - rejected due to slower onboarding and weaker discoverability.
2. Move all canonical specs to repo root - rejected to avoid fragmented ownership and duplicated content.

## Related Specs

- `AGENTS.md`
- `docs/spec-index.md`
- `docs/contract-guardrails.md`
- `docs/spec-lifecycle.md`
