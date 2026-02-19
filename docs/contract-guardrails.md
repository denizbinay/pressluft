Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: contracts/openapi.yaml, docs/data-model.md, docs/state-machines.md, docs/error-codes.md, docs/job-types.md
Supersedes: none

# Contract Guardrails

This document defines hard rules to prevent API/schema drift during implementation.

## OpenAPI-First Rule

- Any API behavior change starts with `contracts/openapi.yaml`.
- `docs/api-contract.md` is explanatory and must be updated in the same PR when behavior changes.
- Feature specs in `docs/features/` must list explicit contract impact.

## Enum and State Alignment

- Enum/state values must stay aligned across:
  - `contracts/openapi.yaml`
  - `docs/data-model.md`
  - `docs/state-machines.md`
- No new enum value is allowed unless all three sources are updated in one change.

## Error Contract

- All non-2xx responses use the shared `Error` schema shape.
- Non-generic failure paths require stable, documented `code` values.
- Do not expose raw infrastructure command output directly in API errors.
- API and job error codes must be registered in `docs/error-codes.md`.

## Job Type Contract

- Async mutation endpoints must map to exactly one registered job type.
- Registered job types are canonical in `docs/job-types.md`.
- Node queries are explicit synchronous exceptions and do not create jobs.

## Breaking-Change Policy (MVP)

- Breaking contract changes are disallowed for v0.1 unless explicitly approved in spec review.
- If unavoidable, include migration strategy and compatibility notes in the same PR.

## Required Change Checklist

1. Feature spec exists and declares in-scope paths.
2. OpenAPI updated first.
3. API doc updated in same PR.
4. Enum/state alignment validated.
5. Error codes and job types updated in registries when changed.
6. Acceptance criteria mapped to verification commands/tests.

## Contract Freshness Rule

When an endpoint behavior changes, the same PR must update all impacted artifacts:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md` (if endpoint ownership/execution changes)
- `docs/error-codes.md` (if error code surface changes)
- `docs/job-types.md` (if async job type surface changes)
