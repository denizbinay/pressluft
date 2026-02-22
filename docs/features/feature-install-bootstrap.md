Status: deprecated
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md
Supersedes: none

# FEATURE: install-bootstrap

Deprecated for Wave 5 provider-first execution. Replaced by:

- `docs/features/feature-wave5-provider-first-node-acquisition.md`
- `docs/features/feature-provider-connections.md`
- `docs/features/feature-hetzner-node-provider.md`

## Problem

Operators need a zero-manual setup path that installs Pressluft and registers the first node deterministically, while preserving explicit node acquisition semantics (`local` vs `remote`) with no hidden fallback.

## Scope

- In scope:
  - Define and implement `install.sh` bootstrap contract for Ubuntu 24.04.
  - Install control-plane prerequisites and Pressluft service.
  - Trigger first-node provisioning and registration flow using explicit node source selection.
  - Ensure local source bootstrap submits async node-create and waits on job/readiness outcomes instead of relying on synchronous provider completion.
  - Require local provider bootstrap to follow provider-equivalent lifecycle (`create/start VM -> inject Pressluft-managed SSH key -> provision over SSH`).
  - Define local acquisition output contract (`hostname`, `public_ip`, `ssh_user`, `ssh_port`, optional `ssh_private_key_path`) for downstream shared provisioning/readiness flow.
  - Use `multipass` as the canonical local acquisition backend for Wave 5.11 bootstrap and return deterministic capability errors when unavailable.
  - Prohibit dependence on provider-internal daemon key material for success-path SSH authentication.
  - Validate idempotent rerun behavior.
- Out of scope:
  - Non-Ubuntu installation flows.
  - Managed-cloud marketplace installers.

## Allowed Change Paths

- `install.sh`
- `internal/bootstrap/**`
- `internal/nodes/**`
- `internal/jobs/**`
- `ansible/playbooks/node-provision.yml`
- `docs/technical-architecture.md`
- `docs/provisioning-spec.md`
- `docs/features/feature-install-bootstrap.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/technical-architecture.md`
- `docs/provisioning-spec.md`
- `docs/ansible-execution.md`

## Acceptance Criteria

1. Running `install.sh` on a fresh Ubuntu 24.04 host installs a runnable Pressluft control plane.
2. First node is registered and enters expected status progression through provisioning after explicit source selection.
3. Local and remote bootstrap paths share the same asynchronous provisioning and readiness pipeline after node acquisition request acceptance.
4. Bootstrap reruns are idempotent and do not create duplicate node records or conflicting services.
5. Bootstrap failures surface deterministic error output with safe rollback guidance.
6. Local acquisition capability failures are explicit and do not trigger remote fallback or host-local runtime shortcuts.
7. Local bootstrap success path uses Pressluft-managed SSH key generation/injection and remains independent of provider-internal private key paths.

## Scenarios (WHEN/THEN)

1. WHEN `install.sh` runs on a fresh Ubuntu 24.04 host THEN Pressluft installs successfully and the control plane starts.
2. WHEN operator selects local node creation THEN bootstrap acquires a local node target and runs standard provisioning without switching to remote automatically.
3. WHEN operator selects remote node creation THEN bootstrap requires explicit remote inputs and does not auto-fallback from local-node failures.
4. WHEN bootstrap is re-run with the same inputs THEN existing services and node records remain stable without duplication.
5. WHEN bootstrap fails mid-sequence THEN partial changes are recoverable with the documented rollback sequence.
6. WHEN local acquisition capability is unavailable THEN bootstrap exits with deterministic remediation guidance and no alternate source fallback.
7. WHEN local provider VM is available but SSH key injection cannot be completed THEN bootstrap reports deterministic failure and does not continue to provisioning with provider-internal key assumptions.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`
- Required tests:
  - Bootstrap flow tests for first-node registration.
  - Idempotency tests for repeated install/bootstrap runs.

## Risks and Rollback

- Risk: partial bootstrap can leave service and node state inconsistent.
- Rollback: stop Pressluft service, remove partial artifacts, and rerun bootstrap from clean checkpoint.
