Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, contracts/openapi.yaml, docs/api-contract.md, docs/contract-guardrails.md, docs/contract-traceability.md
Supersedes: none

# CONTRACTS Router

This file is a top-level routing entry point for API and contract authority.

Canonical sources:

- Machine-readable API contract: `contracts/openapi.yaml`
- Contract guidance: `docs/api-contract.md`
- Contract safety rules: `docs/contract-guardrails.md`
- Endpoint ownership map: `docs/contract-traceability.md`
- Error code registry: `docs/error-codes.md`
- Job type registry: `docs/job-types.md`

Implementation rule:

- API behavior changes must start with `contracts/openapi.yaml` and remain synchronized with supporting docs.
