Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/health-checks.md, docs/job-execution.md, docs/job-types.md, docs/ansible-execution.md, docs/state-machines.md
Supersedes: none

# FEATURE: health-checks-and-rollback

## Problem

Operators need automatic safety gates after deploy, restore, and promotion so unhealthy releases are rolled back immediately.

## Scope

- In scope:
  - Implement `health_check` and `release_rollback` job orchestration.
  - Run required health checks after deploy/restore/promotion.
  - Trigger rollback automatically on health-check failure.
  - Persist release health status transitions and job outcomes transactionally.
- Out of scope:
  - Synthetic uptime monitoring outside job-triggered checks.
  - Multi-stage canary rollback policies.

## Allowed Change Paths

- `internal/jobs/**`
- `internal/environments/**`
- `internal/releases/**`
- `internal/store/**`
- `ansible/playbooks/health-check.yml`
- `ansible/playbooks/release-rollback.yml`
- `docs/health-checks.md`
- `docs/job-types.md`
- `docs/ansible-execution.md`
- `docs/features/feature-health-checks-and-rollback.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/health-checks.md`
- `docs/job-types.md`
- `docs/ansible-execution.md`

## Acceptance Criteria

1. Deploy, restore, and promote flows trigger `health_check` after mutation completion.
2. Failed health checks automatically enqueue `release_rollback` and mark active release unhealthy.
3. Successful rollback returns environment/site state to `active` with prior release restored.
4. Health-check and rollback failures produce stable structured job errors.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/health-check.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/release-rollback.yml`
- Required tests:
  - Post-mutation health gate orchestration tests.
  - Automatic rollback trigger tests.

## Risks and Rollback

- Risk: false-negative health checks may cause unnecessary rollback.
- Rollback: disable health gate for affected operation behind guarded config until check logic is corrected.
