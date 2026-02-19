Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/security-and-secrets.md, docs/data-model.md, docs/job-execution.md
Supersedes: none

# FEATURE: audit-logging

## Problem

Operators need reliable, queryable audit records for all mutating control-plane actions.

## Scope

- In scope:
  - Define and enforce audit logging contract for all mutating API actions.
  - Persist `audit_logs` rows with stable action/resource/result semantics.
  - Cover both synchronous mutations and async job-triggering mutations.
- Out of scope:
  - External SIEM forwarding.
  - Long-term audit retention policy customization.

## Allowed Change Paths

- `internal/api/**`
- `internal/audit/**`
- `internal/jobs/**`
- `internal/store/**`
- `docs/security-and-secrets.md`
- `docs/data-model.md`
- `docs/features/feature-audit-logging.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `docs/security-and-secrets.md`
- `docs/data-model.md`

## Acceptance Criteria

1. Every mutating API action writes one audit entry with `user_id`, `action`, `resource_type`, `resource_id`, `result`, and timestamp.
2. Async mutation requests write audit entries at request acceptance time and update result semantics on completion path.
3. Audit logging failures do not leak secrets in errors or logs.
4. Magic login remains covered as a synchronous audited action.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Audit write tests across representative mutating endpoints.
  - Failure-path tests for audit persistence behavior.

## Risks and Rollback

- Risk: missing audit writes reduce forensic traceability.
- Rollback: fail closed for audit-critical endpoints or queue compensating audit records where possible.
