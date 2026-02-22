Status: deprecated
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md
Supersedes: none

# FEATURE: wave5-node-acquisition-parity

Deprecated in favor of provider-first acquisition.

Replacement specs:

- `docs/features/feature-wave5-provider-first-node-acquisition.md`
- `docs/features/feature-provider-connections.md`
- `docs/features/feature-hetzner-node-provider.md`

## Problem

Wave 5 runtime behavior is contract-aligned once a node is ready, but local quickstart/development still relies on host-specific prerequisites that are easy to miss. This creates operator friction and risks drift between local quickstart behavior and fresh Ubuntu node behavior.

## Scope

- In scope:
  - Define a single node acquisition contract that feeds the existing provisioning/readiness pipeline.
  - Add operator-visible node creation entry points with two explicit choices: `Create Local Node` and `Create Remote Node`.
  - Enforce no hidden fallback between node types; selected node type must be honored deterministically.
  - Implement local-node quickstart path that acquires a local Ubuntu 24.04 VM target and then provisions it through the same SSH + Ansible flow used for remote nodes.
  - Treat local acquisition as a provider path with the same lifecycle shape as remote/cloud providers (`create VM -> inject Pressluft-managed SSH key -> provision over SSH`).
  - Route local acquisition through asynchronous job execution (`node_provision`) so `POST /api/nodes` responds with accepted job semantics while acquisition/provisioning complete in background.
  - Use `multipass` as the canonical local VM backend for Wave 5.11 and return deterministic capability errors when unavailable.
  - Manage local-provider SSH key lifecycle in Pressluft-owned paths and inject the public key into acquired VM user `authorized_keys` before Ansible provisioning.
  - Prohibit dependency on provider-internal/private key paths (for example daemon-managed keys under root-owned system paths).
  - Require local node records to be created from acquisition output (`hostname`, `public_ip`, `ssh_user`, `ssh_port`, optional `ssh_private_key_path`) rather than seeded loopback defaults.
  - Require readiness probes for acquired local nodes to execute against the node target over SSH, matching remote-node readiness semantics.
  - Prepare remote-node creation contract for later provider integrations while preserving current Wave 5 constraints.
- Out of scope:
  - Per-site containerized WordPress runtimes.
  - Wave 6 deploy/update/rollback behavior.
  - Multi-cloud provider integrations beyond contract shape.
  - Local-host fallback runtime execution that bypasses acquired-node SSH targeting.

## Allowed Change Paths

- `cmd/pressluft/**`
- `internal/bootstrap/**`
- `internal/nodes/**`
- `internal/api/**`
- `internal/devserver/**`
- `internal/jobs/**`
- `internal/ssh/**`
- `ansible/playbooks/node-provision.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/testing.md`
- `docs/features/feature-wave5-node-acquisition-parity.md`
- `docs/features/feature-install-bootstrap.md`

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

1. Node creation UI exposes exactly two operator actions: `Create Local Node` and `Create Remote Node`.
2. `Create Local Node` immediately returns accepted-job semantics and provisions a local Ubuntu VM target asynchronously through the same node-provisioning flow used for remote nodes.
3. `Create Remote Node` uses explicit remote host inputs and does not auto-fallback from local creation failures.
4. Readiness checks and failure reason codes remain the same regardless of node acquisition source.
5. Wave 5 smokes can run against an acquired local node without requiring manual runtime prerequisite installs on the developer host.
6. Dev runtime does not auto-seed a local loopback node for success-path verification.
7. Local acquisition failures return deterministic local capability semantics as terminal job outcomes with stable error codes and no implicit source fallback.
8. Local provider success path does not require reading provider-internal SSH private keys; Pressluft-managed key generation/injection is used instead.

## Scenarios (WHEN/THEN)

1. WHEN an operator clicks `Create Local Node` THEN Pressluft acquires a local VM node and proceeds through standard provision/readiness transitions.
2. WHEN local-node acquisition fails THEN Pressluft returns deterministic local-acquisition failure semantics and does not create a remote node implicitly.
3. WHEN local-node acquisition is still preparing THEN Pressluft keeps `POST /api/nodes` accepted and resolves readiness through job retries/state transitions rather than immediate API conflict.
4. WHEN an operator clicks `Create Remote Node` THEN Pressluft creates only a remote node path and validates remote connectivity requirements explicitly.
5. WHEN `multipass` is unavailable THEN local create job fails with deterministic capability error output and no hidden fallback behavior.
6. WHEN provider VM creation succeeds but SSH key injection fails THEN job fails with deterministic local acquisition/provisioning failure semantics and no fallback.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `bash scripts/smoke-create-site-preview.sh`
  - `bash scripts/smoke-site-clone-preview.sh`
  - `bash scripts/smoke-backup-restore.sh`
- Required tests:
  - Node creation API/UI tests for local-vs-remote button actions and no-fallback behavior.
  - Provision/readiness tests proving parity between local and remote node acquisition paths.
  - Local provider tests proving Pressluft-managed SSH key generation/reuse and authorized-keys injection semantics.
  - Wave 5 smoke assertions on acquired local node runtime.

## Risks and Rollback

- Risk: local VM acquisition tooling can vary by host platform and introduce setup friction.
- Rollback: keep remote-node path operational and gate local-node acquisition behind an explicit capability flag while preserving no-fallback semantics.
