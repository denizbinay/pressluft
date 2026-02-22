Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/ansible-execution.md, docs/features/feature-wave5-provider-first-node-acquisition.md, docs/features/feature-provider-connections.md, docs/features/feature-hetzner-sdk-integration.md
Supersedes: none

# FEATURE: hetzner-node-provider

## Problem

Wave 5 requires one concrete provider implementation to unblock deterministic node acquisition and closeout smokes.

## Scope

- In scope:
  - Implement Hetzner provider adapter for node acquisition using `github.com/hetznercloud/hcloud-go`.
  - Use provider lifecycle `create server -> poll action -> fetch server -> provision`.
  - Register/reuse Pressluft-managed SSH public key with Hetzner.
  - Map Hetzner failures into stable provider error classes.
  - Preserve existing node provisioning/readiness invariants after acquisition output is produced.
- Out of scope:
  - Load balancers, volumes, or non-server Hetzner resources.
  - Cross-provider failover logic.

## Allowed Change Paths

- `internal/providers/hetzner/**`
- `go.mod`
- `go.sum`
- `internal/nodes/**`
- `internal/jobs/**`
- `internal/api/**`
- `docs/error-codes.md`
- `docs/testing.md`
- `docs/features/feature-hetzner-node-provider.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/error-codes.md`

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli

## Acceptance Criteria

1. Hetzner-backed node create requests return `202` and complete asynchronously via `node_provision`.
2. Acquisition lifecycle captures server id/action id and converges to a provisionable SSH target via SDK-backed operations.
3. Hetzner API and provisioning failures surface deterministic, provider-scoped error codes.
4. Existing provisioning and readiness checks run unchanged after acquisition handoff.

## Scenarios (WHEN/THEN)

1. WHEN Hetzner node create succeeds THEN node transitions through queued/running/succeeded with valid target host data.
2. WHEN Hetzner action polling times out THEN job fails with stable provider timeout error.
3. WHEN provisioning fails after successful provider acquisition THEN failure remains classified as provisioning/runtime failure, not provider create failure.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Hetzner adapter unit tests for success and failure mapping.
  - Node-provision integration tests for provider acquisition handoff.

## Risks and Rollback

- Risk: provider API drift may break request/response assumptions.
- Rollback: pin to last known-good adapter behavior and keep compatibility shims for updated provider fields.
