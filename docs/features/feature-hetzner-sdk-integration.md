Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/features/feature-provider-connections.md, docs/features/feature-hetzner-node-provider.md, docs/security-and-secrets.md
Supersedes: none

# FEATURE: hetzner-sdk-integration

## Problem

Wave 5.11 currently uses custom Hetzner API request logic that increases drift risk, duplicates provider-client behavior, and has already produced credential validation mismatches.

## Scope

- In scope:
  - Standardize Hetzner API communication on `github.com/hetznercloud/hcloud-go`.
  - Add a provider credential health-check flow that validates real token usability instead of token-prefix heuristics.
  - Keep provider connection status model deterministic for `/providers` and `/nodes` operator workflows.
  - Keep `POST /api/nodes` async semantics unchanged (`202` + `node_provision` lifecycle).
  - Preserve stable provider error classification (`PROVIDER_*`) for retries and operator guidance.
- Out of scope:
  - Replacing Ansible provisioning stage after provider acquisition handoff.
  - Multi-provider runtime implementations beyond Hetzner.

## Allowed Change Paths

- `go.mod`
- `go.sum`
- `internal/providers/**`
- `internal/nodes/**`
- `internal/api/**`
- `internal/devserver/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/error-codes.md`
- `docs/testing.md`
- `docs/ui-flows.md`
- `docs/features/feature-hetzner-sdk-integration.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/error-codes.md`
- `docs/testing.md`

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli

## Acceptance Criteria

1. Hetzner provider adapter uses `hcloud-go` for server create/poll/fetch and managed SSH key registration/reuse.
2. Provider connect flow uses live credential validation and no longer degrades based on `hcloud_` prefix checks.
3. `/providers` and `/nodes` dashboards show status/guidance aligned with real provider health outcomes.
4. `POST /api/nodes` accepts only connected providers and retains deterministic `202` accepted-job behavior.
5. Wave 5 smokes (`clone`, `backup/restore`) remain mandatory and pass on provider-acquired nodes using validated credentials.

## Scenarios (WHEN/THEN)

1. WHEN operator submits a valid bearer token THEN provider status becomes `connected` after live health check.
2. WHEN token is invalid/insufficient THEN provider status remains non-healthy with deterministic remediation guidance.
3. WHEN `node_provision` runs THEN provider acquisition succeeds or fails with stable provider-scoped error classes.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `bash scripts/smoke-site-clone-preview.sh`
  - `bash scripts/smoke-backup-restore.sh`
- Required tests:
  - Provider service tests for credential validation and status transitions.
  - Hetzner adapter tests for `hcloud-go` acquisition lifecycle and deterministic failure mapping.
  - Dashboard tests for provider-token guidance messaging.

## Risks and Rollback

- Risk: SDK upgrade introduces API behavior differences from current adapter assumptions.
- Rollback: pin to last known-good SDK version and preserve provider status + error contract compatibility.
