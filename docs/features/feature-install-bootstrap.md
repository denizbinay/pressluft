Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/provisioning-spec.md, docs/ansible-execution.md
Supersedes: none

# FEATURE: install-bootstrap

## Problem

Operators need a zero-manual setup path that installs Pressluft and registers the first node deterministically.

## Scope

- In scope:
  - Define and implement `install.sh` bootstrap contract for Ubuntu 24.04.
  - Install control-plane prerequisites and Pressluft service.
  - Trigger first-node provisioning and registration flow.
  - Validate idempotent rerun behavior.
- Out of scope:
  - Non-Ubuntu installation flows.
  - Managed-cloud marketplace installers.

## Allowed Change Paths

- `install.sh`
- `go.mod`
- `go.sum`
- `cmd/pressluft/**`
- `internal/bootstrap/**`
- `internal/nodes/**`
- `internal/jobs/**`
- `internal/store/**`
- `migrations/migrate.go`
- `.github/workflows/ci.yml`
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
2. First local node is registered and enters expected status progression through provisioning.
3. Bootstrap reruns are idempotent and do not create duplicate node records or conflicting services.
4. Bootstrap failures surface deterministic error output with safe rollback guidance.

## Scenarios (WHEN/THEN)

1. WHEN `install.sh` runs on a fresh Ubuntu 24.04 host THEN Pressluft installs successfully and the control plane starts.
2. WHEN bootstrap is re-run with the same inputs THEN existing services and node records remain stable without duplication.
3. WHEN bootstrap fails mid-sequence THEN partial changes are recoverable with the documented rollback sequence.

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
