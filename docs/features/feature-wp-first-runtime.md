Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/job-execution.md, docs/ansible-execution.md, docs/provisioning-spec.md, docs/domain-and-routing.md, docs/ui-flows.md
Supersedes: none

# FEATURE: wp-first-runtime

## Problem

Operators can currently create records, enqueue jobs, and navigate dashboard flows, but they still need a deterministic path where creating a site yields a reachable WordPress runtime URL on a valid node target.

## Scope

- In scope:
  - Define self-node runtime behavior for local/WSL2 execution.
  - Ensure `site_create` and `env_create` mutation execution provisions a runnable WordPress stack.
  - Align job lifecycle semantics so mutation success requires runtime reachability validation.
  - Add an end-to-end smoke contract for create-site to reachable preview URL.
- Out of scope:
  - New public API endpoints.
  - Multi-node traffic orchestration.
  - Production-grade HA/load-balancing concerns.

## Allowed Change Paths

- `internal/jobs/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/nodes/**`
- `internal/store/**`
- `internal/api/**`
- `internal/devserver/**`
- `ansible/playbooks/**`
- `ansible/roles/**`
- `scripts/**`
- `docs/provisioning-spec.md`
- `docs/ansible-execution.md`
- `docs/ui-flows.md`
- `docs/features/feature-wp-first-runtime.md`
- `changes/wp-first-runtime/**`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/ansible-execution.md`
- `docs/provisioning-spec.md`
- `docs/ui-flows.md`

## Acceptance Criteria

1. On a valid self-node target, creating a site results in a reachable preview URL that serves WordPress.
2. `site_create` and `env_create` job success is emitted only after runtime reachability checks pass.
3. Runtime provisioning failures produce stable job failure states and operator-visible error context.
4. Wave 5.5 can be verified end-to-end using local/WSL2 baseline without manual post-fix commands, including active worker execution in the `pressluft dev` runtime path.

## Scenarios (WHEN/THEN)

1. WHEN the operator creates a site on self-node THEN the control plane executes queue + Ansible mutation flow and returns a preview URL that responds successfully.
2. WHEN runtime provisioning fails at any stage THEN the job transitions to failed with stable error semantics and no false-positive active status.
3. WHEN clone/staging environment creation succeeds THEN the generated preview URL is reachable and maps to the new environment runtime.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`
- Required tests:
  - Runtime-oriented mutation lifecycle tests in `internal/jobs/**` and `internal/sites/**`.
  - End-to-end smoke verification for create-site to preview URL reachability.

## Risks and Rollback

- Risk: local/WSL2 networking and DNS behavior may differ from remote node assumptions.
- Rollback: keep existing queue/state semantics, gate new reachability requirement behind controlled rollout while preserving previous stable behavior.
