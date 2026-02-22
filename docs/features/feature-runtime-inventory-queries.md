Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/security-and-secrets.md, docs/api-contract.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: runtime-inventory-queries

## Problem

Operators need live runtime inventory data to trust dashboard decisions. Current surfaces do not provide a direct nodes list endpoint or a live WordPress version query for environments.

## Scope

- In scope:
  - Implement `GET /api/nodes` for authenticated node visibility.
  - Implement `GET /api/environments/{id}/wordpress-version` as a synchronous node query.
  - Enforce single SSH command and 10-second timeout for runtime inventory query.
  - Return stable error codes for not found, inactive environment, node unreachable, and WordPress command failure.
  - Add audit entries for inventory query invocations.
- Out of scope:
  - Persisted inventory snapshots in DB.
  - Background polling jobs for runtime inventory.
  - Plugin/theme/version bundle inventory beyond WordPress core version string.

## Allowed Change Paths

- `internal/api/**`
- `internal/ssh/**`
- `internal/nodes/**`
- `internal/environments/**`
- `internal/audit/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/features/feature-runtime-inventory-queries.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`

## Acceptance Criteria

1. `GET /api/nodes` returns all registered nodes with status and provider/source marker.
2. `GET /api/environments/{id}/wordpress-version` returns `200` with `{ environment_id, wordpress_version, queried_at }` for active environments on reachable nodes.
3. `GET /api/environments/{id}/wordpress-version` returns stable errors for `404` not found, `409` environment not active, and `502` node/WP command failure classes.
4. Node query executes as a single SSH command with 10-second hard timeout and does not enqueue jobs.
5. Query requests are audit logged with stable action/resource semantics.

## Scenarios (WHEN/THEN)

1. WHEN the dashboard loads nodes THEN it reads from `GET /api/nodes` and displays provider-backed node readiness.
2. WHEN the dashboard renders site rows THEN it resolves WordPress version via `GET /api/environments/{id}/wordpress-version` for site production environment.
3. WHEN node query fails THEN operator sees stable error code and existing data remains unchanged.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - API tests for `GET /api/nodes` and `GET /api/environments/{id}/wordpress-version` success/failure responses.
  - SSH timeout test for inventory query (10-second cap).
  - Audit log tests for inventory query success/failure.

## Risks and Rollback

- Risk: synchronous inventory query can add latency to dashboard rendering.
- Rollback: keep `GET /api/nodes` and temporarily gate WordPress version query behind explicit operator action in UI.
