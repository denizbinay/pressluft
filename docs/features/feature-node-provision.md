Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/provisioning-spec.md, docs/ansible-execution.md, docs/security-and-secrets.md
Supersedes: none

# FEATURE: node-provision

## Problem

The control plane must bootstrap and harden Ubuntu nodes deterministically before hosting any environments.

## Scope

- In scope:
  - Implement `node_provision` job dispatch and state handling.
  - For provider-acquired nodes, run acquisition and provisioning in the same async `node_provision` lifecycle.
  - Require provider lifecycle completion (`create server + poll action + fetch server target`) before Ansible provisioning executes.
  - Execute `ansible/playbooks/node-provision.yml` with DB-derived inventory and vars.
  - Validate required runtime components, TLS prerequisites, and security baseline.
- Out of scope:
  - Multi-OS support.
  - Ad hoc shell provisioning outside Ansible.

## Allowed Change Paths

- `internal/jobs/**`
- `internal/store/**`
- `internal/nodes/**`
- `ansible/playbooks/node-provision.yml`
- `ansible/roles/**`
- `docs/ansible-execution.md`
- `docs/provisioning-spec.md`
- `docs/features/feature-node-provision.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/ansible-execution.md`
- `docs/provisioning-spec.md`

## Acceptance Criteria

1. Node provisioning runs only through job queue and Ansible.
2. Provider acquisition preparation states are handled as retryable job outcomes, not synchronous API failures.
3. Provider-side SSH key registration/injection is deterministic and idempotent across retries.
4. Provisioning is idempotent across repeated runs.
5. Node status and job status transition transactionally for success and failure.
6. Provisioning failure surfaces structured error code and truncated output.

## Scenarios (WHEN/THEN)

1. WHEN `node_provision` is requested THEN the control plane enqueues a job and executes acquisition/provisioning steps for the selected provider.
2. WHEN provisioning is retried for an already provisioned node THEN the run is idempotent and does not break node state.
3. WHEN provider acquisition reports transient preparation state THEN the job requeues with deterministic retry behavior.
4. WHEN provider-side SSH key registration/injection fails THEN job fails with deterministic provider acquisition/provisioning error semantics and no fallback path.
5. WHEN Ansible fails or times out THEN job and node status transitions remain transactional and error codes are stable.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`
- Required tests:
  - Job executor tests for node_provision dispatch.
  - Retry and timeout behavior tests.
  - Provider key-registration/injection and idempotent retry tests.

## Risks and Rollback

- Risk: non-idempotent task can break reruns.
- Rollback: revert playbook change set and rerun known-good provisioning version.
