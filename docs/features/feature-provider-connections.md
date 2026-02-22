Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/security-and-secrets.md, docs/features/feature-wave5-provider-first-node-acquisition.md, docs/features/feature-hetzner-sdk-integration.md
Supersedes: none

# FEATURE: provider-connections

## Problem

Wave 5 has no first-class provider connection model for storing credentials, surfacing connection health, and driving provider-backed node creation.

## Scope

- In scope:
  - Define provider connection state model (`connected`, `degraded`, `disconnected`) for dashboard/API use.
  - Add `/providers` dashboard surface for connecting and inspecting providers.
  - Store provider credentials as persisted secrets per security constraints.
  - Validate Hetzner credentials with live provider check (no token-prefix heuristics).
  - Provide provider capability metadata needed by node creation controls.
  - Keep model extensible for additional providers.
- Out of scope:
  - Provider billing/cost management.
  - Multi-tenant credential isolation.

## Allowed Change Paths

- `internal/providers/**`
- `internal/api/**`
- `internal/devserver/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/ui-flows.md`
- `docs/security-and-secrets.md`
- `docs/testing.md`
- `docs/features/feature-provider-connections.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/ui-flows.md`
- `docs/security-and-secrets.md`
- `docs/testing.md`

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli

## Acceptance Criteria

1. `/providers` route exists and is linked from dashboard shell.
2. Operator can connect Hetzner via UI with persisted secret handling and live credential validation.
3. Provider status and guidance are visible and actionable before node creation.
4. Provider model supports adding another provider without breaking existing contracts.

## Scenarios (WHEN/THEN)

1. WHEN no providers are connected THEN `/providers` shows explicit setup guidance and blocks provider-backed node create.
2. WHEN Hetzner credentials are saved THEN provider status reflects live provider health for node workflows.
3. WHEN provider status degrades THEN `/providers` and `/nodes` show deterministic remediation guidance.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Provider connection API tests.
  - Dashboard `/providers` route and status rendering tests.

## Risks and Rollback

- Risk: secret handling mistakes can leak provider credentials.
- Rollback: disable provider mutation endpoints and keep provider read-only status until secure path is restored.
