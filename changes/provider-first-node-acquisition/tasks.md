# Tasks: provider-first-node-acquisition

## Wave Plan

- Wave 0 (planning): replace node-acquisition parity packet with provider-first packet and update Wave 5.11 backlog.
- Wave 1 (implementation): provider connections + provider-only node create + Hetzner adapter + cleanup.
- Merge point(s): MP1.5 (Wave 5.11 complete before Wave 6).

## Atomic Tasks

- [x] T1: Replace Wave 5.11 planning/spec scope with provider-first node acquisition.
  - Depends on: none
  - Paths: `PLAN.md`, `PROGRESS.md`, `docs/features/*`, `changes/provider-first-node-acquisition/*`
  - Verification: `bash scripts/check-readiness.sh`
- [x] T2: Implement provider connection control plane and `/providers` route with persisted secret handling.
  - Depends on: T1
  - Paths: `internal/providers/**`, `internal/api/**`, `internal/devserver/**`, `contracts/openapi.yaml`, `docs/api-contract.md`
  - Verification: `go test ./internal/... -v`
- [x] T3: Replace node-create contract with provider-backed request shape only.
  - Depends on: T2
  - Paths: `contracts/openapi.yaml`, `internal/api/**`, `internal/nodes/**`, `docs/contract-traceability.md`, `docs/error-codes.md`
  - Verification: `go test ./internal/... -v`, `bash scripts/check-readiness.sh`
- [x] T4: Implement Hetzner async acquisition lifecycle in `node_provision` and handoff to existing provisioning/readiness.
  - Depends on: T3
  - Paths: `internal/providers/hetzner/**`, `internal/nodes/**`, `internal/jobs/**`, `docs/features/feature-hetzner-node-provider.md`
  - Verification: `go test ./internal/... -v`
- [x] T5: Remove obsolete local/manual/self-node acquisition code, docs references, and regression fixtures.
  - Depends on: T4
  - Paths: `internal/bootstrap/**`, `internal/devserver/**`, `scripts/**`, `docs/**`
  - Verification: `go test ./internal/... -v`, `bash scripts/check-readiness.sh`
- [ ] T6: Remove static token-prefix provider checks and use live provider health validation for credential status.
  - Depends on: T5
  - Paths: `internal/providers/**`, `internal/api/**`, `internal/devserver/**`, `docs/features/*`
  - Verification: `go test ./internal/... -v`
- [ ] T7: Migrate Hetzner acquisition adapter to `hcloud-go` and preserve deterministic `PROVIDER_*` mapping.
  - Depends on: T6
  - Paths: `go.mod`, `go.sum`, `internal/providers/hetzner/**`, `internal/nodes/**`, `docs/error-codes.md`
  - Verification: `go test ./internal/... -v`, `go vet ./...`
- [ ] T8: Align `/providers` and `/nodes` UX/API guidance with bearer-token semantics.
  - Depends on: T6
  - Paths: `internal/devserver/**`, `contracts/openapi.yaml`, `docs/api-contract.md`, `docs/ui-flows.md`, `docs/features/*`
  - Verification: `go test ./internal/... -v`, `bash scripts/check-readiness.sh`
- [ ] T9: Extend Wave 5 smoke/regression coverage for SDK-backed provider flows and deterministic diagnostics.
  - Depends on: T7, T8
  - Paths: `scripts/**`, `docs/testing.md`, `docs/features/*`, `PROGRESS.md`
  - Verification: `bash scripts/smoke-site-clone-preview.sh`, `bash scripts/smoke-backup-restore.sh`
- [ ] T10: Run provider-backed Wave 5 closeout evidence and close pending smoke-linked tasks.
  - Depends on: T9
  - Paths: `PROGRESS.md`, `PLAN.md`
  - Verification: `bash scripts/smoke-site-clone-preview.sh`, `bash scripts/smoke-backup-restore.sh`

## Acceptance Mapping

- AC1 -> T3/T4 contract + queue tests.
- AC2 -> T2 provider connection API/UI tests.
- AC3 -> T4 provider lifecycle integration tests.
- AC4 -> T5 cleanup verification.
- AC5 -> T9/T10 smoke outputs and readiness evidence.
