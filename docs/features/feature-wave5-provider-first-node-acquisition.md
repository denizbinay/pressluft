Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/job-execution.md, docs/ansible-execution.md, docs/features/feature-node-provision.md, docs/features/feature-hetzner-sdk-integration.md
Supersedes: docs/features/feature-wave5-node-acquisition-parity.md, docs/features/feature-install-bootstrap.md

# FEATURE: wave5-provider-first-node-acquisition

## Problem

Wave 5 node acquisition still carries local (`multipass`) and manual remote SSH creation paths that introduce host-specific drift, ambiguous ownership boundaries, and brittle runtime assumptions.

## Scope

- In scope:
  - Replace Wave 5 node acquisition with provider-backed semantics only.
  - Remove local node create and manual remote SSH node create from Wave 5 operator flows.
  - Require Hetzner provider communication through `hcloud-go` in active runtime path.
  - Keep async accepted-job behavior (`202`) for node creation while provider acquisition and provisioning execute in background.
  - Keep provisioning/readiness execution model unchanged after provider acquisition completes (job queue + SSH + Ansible).
  - Add explicit cleanup requirements for obsolete local/manual/self-node acquisition paths.
- Out of scope:
  - Multi-provider orchestration beyond one active provider implementation in Wave 5.
  - Wave 6 deployment/update behavior.

## Allowed Change Paths

- `contracts/openapi.yaml`
- `internal/api/**`
- `internal/nodes/**`
- `internal/jobs/**`
- `internal/devserver/**`
- `internal/bootstrap/**`
- `go.mod`
- `go.sum`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/testing.md`

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli
- `docs/features/feature-wave5-provider-first-node-acquisition.md`
- `docs/features/feature-provider-connections.md`
- `docs/features/feature-hetzner-node-provider.md`
- `docs/features/feature-node-provision.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/testing.md`

## Acceptance Criteria

1. `/nodes` no longer exposes `Create Local Node` or manual remote SSH creation inputs.
2. Node creation API contracts are provider-backed only and remain async accepted-job semantics.
3. Wave 5 create-site/clone/backup-restore flows succeed on provider-acquired nodes.
4. Obsolete local/manual/self-node acquisition code and docs are removed or marked deprecated.
5. Provider selection/model is extensible and does not require a redesign for adding the next provider.
6. Provider credential health no longer depends on static token-prefix checks.

## Scenarios (WHEN/THEN)

1. WHEN operator creates a node THEN request targets a connected provider and returns `202` with job identity.
2. WHEN provider acquisition fails THEN job fails with deterministic provider-classified errors and no local/manual fallback path.
3. WHEN provider acquisition succeeds THEN provisioning/readiness transitions execute through the existing queue + Ansible + SSH invariants.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `bash scripts/smoke-site-clone-preview.sh`
  - `bash scripts/smoke-backup-restore.sh`
- Required tests:
  - Node API/UI contract tests for provider-only create semantics.
  - Queue/provisioning tests proving provider acquisition handoff.
  - Regression tests proving local/manual/self-node acquisition paths are removed.

## Risks and Rollback

- Risk: provider API credential/setup errors can block node acquisition.
- Rollback: keep provider abstraction and fall back to previous stable provider contract version without re-introducing local/manual create paths.
