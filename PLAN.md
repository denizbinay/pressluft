# Pressluft MVP Plan

Status: active
Owner: platform
Last Updated: 2026-02-20

## Preconditions (Must Stay Green)

1. Spec and contract readiness checks pass (`docs/pre-plan-readiness.md`).
2. Baseline executable schema exists under `migrations/` and applies successfully.
3. Contract authority remains OpenAPI-first (`contracts/openapi.yaml`).

## Prioritized Implementation Order

1. `docs/features/feature-install-bootstrap.md`
2. `docs/features/feature-node-provision.md`
3. `docs/features/feature-auth-session.md`
4. `docs/features/feature-site-create.md`
5. `docs/features/feature-environment-create-clone.md`
6. `docs/features/feature-environment-deploy-updates.md`
7. `docs/features/feature-health-checks-and-rollback.md`
8. `docs/features/feature-backups.md`
9. `docs/features/feature-environment-restore.md`
10. `docs/features/feature-promotion-drift.md`
11. `docs/features/feature-domains-and-tls.md`
12. `docs/features/feature-cache-controls.md`
13. `docs/features/feature-magic-login.md`
14. `docs/features/feature-jobs-and-metrics.md`
15. `docs/features/feature-audit-logging.md`
16. `docs/features/feature-backup-retention-cleanup.md`
17. `docs/features/feature-site-import.md`
18. `docs/features/feature-settings-domain-config.md`
19. `docs/features/feature-job-control.md`

## Dependency Mapping

### Schema Dependencies

- `docs/data-model.md` -> `migrations/` baseline + forward migrations.
- State and enum changes must be synchronized with:
  - `docs/state-machines.md`
  - `contracts/openapi.yaml`

### Contract Dependencies

- `contracts/openapi.yaml` is canonical for API behavior.
- Endpoint ownership must remain 1:1 in `docs/contract-traceability.md`.
- Async endpoints must map to one canonical type in `docs/job-types.md`.
- Error code surfaces must stay registered in `docs/error-codes.md`.

### Test and Verification Dependencies

- Before commit:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
- Before PR:
  - `go test ./internal/... -v`
- Frontend gate (when `web/` exists):
  - `cd web && pnpm install`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`

## Iteration Structure

### Iteration 0 - Repository Bootstrap Readiness

- Establish runnable Go module and command surfaces (`cmd/pressluft`, `internal/**`).
- Ensure migration runner executes on a clean DB.
- Add minimum CI wiring for build, vet, and tests.

### Iteration 1 - Core Runtime and Job Foundations

- Auth session, DB store layer, state/version guards, and job queue lock invariants.
- Node provision path through queue + Ansible contract.

### Iteration 2 - Site and Environment Lifecycle

- Site create, env create/clone, deploy/updates, health gates, and rollback.

### Iteration 3 - Backup, Restore, and Promotion Safety

- Backup create/list, restore, drift check, and promotion gates.

### Iteration 4 - Domain, Cache, and Operator Workflows

- Domain add/remove, cache toggle/purge, settings domain config behavior.

### Iteration 5 - Observability and Control Surfaces

- Jobs and metrics API, audit logging, backup cleanup, and admin job control.

## Planning Rules

- No implementation starts without its owning feature spec.
- No schema change ships without a migration.
- No API behavior change ships without OpenAPI update in the same change.
