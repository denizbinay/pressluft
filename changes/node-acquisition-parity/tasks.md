# Tasks: node-acquisition-parity

## Wave Plan

- Wave 0 (foundation): finalize feature specs + contracts for explicit node creation semantics.
- Wave 1 (dependent): implement local-node acquisition, remote-node scaffolding, and parity verification.
- Merge point(s): MP1.5 (Wave 5.11 complete before Wave 6).

## Atomic Tasks

- [x] T1: Author feature specs and planning artifacts for node acquisition parity and no-fallback semantics.
  - Depends on: none
  - Paths: `PLAN.md`, `PROGRESS.md`, `docs/features/feature-wave5-node-acquisition-parity.md`, `docs/features/feature-install-bootstrap.md`, `docs/ui-flows.md`, `changes/node-acquisition-parity/*`
  - Verification: `bash scripts/check-readiness.sh`
- [x] T2: Define and implement node creation contract with explicit source type and no implicit fallback.
  - Depends on: T1
  - Paths: `contracts/openapi.yaml`, `docs/api-contract.md`, `docs/contract-traceability.md`, `docs/error-codes.md`, `internal/api/**`, `internal/nodes/**`
  - Verification: `go test ./internal/... -v`
- [ ] T3: Implement async local-node acquisition adapter and wire it to existing provisioning/readiness flow.
  - Depends on: T2
  - Paths: `internal/bootstrap/**`, `internal/nodes/**`, `internal/jobs/**`, `ansible/playbooks/node-provision.yml`, `docs/testing.md`
  - Verification: `go test ./internal/... -v`, `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`
- [ ] T3a: Implement `multipass`-backed local acquisition provider lifecycle with provider-equivalent semantics (`create/start VM -> inject Pressluft-managed SSH key -> return target`).
  - Depends on: T3
  - Paths: `internal/bootstrap/**`, `docs/features/feature-wave5-node-acquisition-parity.md`, `docs/features/feature-install-bootstrap.md`
  - Verification: `go test ./internal/... -v`
- [ ] T3a.1: Add Pressluft-managed SSH key generation/reuse for local provider path and persist returned key path in acquisition target output.
  - Depends on: T3a
  - Paths: `internal/bootstrap/**`, `internal/nodes/**`, `docs/features/feature-wave5-node-acquisition-parity.md`
  - Verification: `go test ./internal/... -v`
- [ ] T3a.2: Inject generated public key into acquired local VM `authorized_keys` via provider command path and enforce idempotent behavior across retries.
  - Depends on: T3a.1
  - Paths: `internal/bootstrap/**`, `internal/nodes/**`, `docs/features/feature-node-provision.md`
  - Verification: `go test ./internal/... -v`
- [ ] T3b: Remove success-path auto-seeded loopback self-node behavior from dev runtime/create-flow path.
  - Depends on: T3a
  - Paths: `internal/devserver/**`, `internal/nodes/**`, `internal/api/**`
  - Verification: `go test ./internal/... -v`
- [ ] T3c: Enforce readiness parity over SSH for acquired local nodes (no control-plane shell fallback for success path).
  - Depends on: T3b
  - Paths: `internal/nodes/**`, `internal/ssh/**`, `internal/api/**`, `docs/features/feature-wave5-runtime-readiness.md`
  - Verification: `go test ./internal/... -v`
- [ ] T3d: Add deterministic failure-semantics coverage for local capability/acquisition/provisioning failures.
  - Depends on: T3c
  - Paths: `internal/bootstrap/**`, `internal/api/**`, `internal/devserver/**`
  - Verification: `go test ./internal/... -v`
- [ ] T3d.1: Add deterministic failure classification for provider key bootstrap/injection errors and ensure they are surfaced as async job outcomes (not API-path conflicts).
  - Depends on: T3d
  - Paths: `internal/bootstrap/**`, `internal/nodes/**`, `docs/error-codes.md`
  - Verification: `go test ./internal/... -v`
- [ ] T3e: Move local acquisition out of synchronous `POST /api/nodes` request path and into `node_provision` worker lifecycle.
  - Depends on: T3d
  - Paths: `internal/api/**`, `internal/nodes/**`, `internal/jobs/**`, `docs/api-contract.md`, `contracts/openapi.yaml`, `docs/error-codes.md`
  - Verification: `go test ./internal/... -v`, `bash scripts/check-readiness.sh`
- [x] T4: Implement `/nodes` create UI controls with two explicit actions and deterministic failure messaging.
  - Depends on: T2
  - Paths: `internal/devserver/**`, `docs/ui-flows.md`
  - Verification: `go test ./internal/... -v`, route smoke checks on `/nodes`
- [ ] T5: Extend Wave 5 smoke/regression coverage to validate async local-node acquisition + create-site/clone/backup-restore parity.
  - Depends on: T3e, T4
  - Paths: `scripts/**`, `internal/devserver/**`, `internal/sites/**`, `internal/environments/**`, `PROGRESS.md`
  - Verification: `bash scripts/smoke-create-site-preview.sh`, `bash scripts/smoke-site-clone-preview.sh`, `bash scripts/smoke-backup-restore.sh`
- [ ] T5a: Prove success-path smoke flows against acquired local node runtime (site create, clone create, backup/restore) with no manual re-submit during provider prep windows.
  - Depends on: T3e, T4, T3a.2
  - Paths: `scripts/**`, `PROGRESS.md`
  - Verification: `bash scripts/smoke-create-site-preview.sh`, `bash scripts/smoke-site-clone-preview.sh`, `bash scripts/smoke-backup-restore.sh`
- [ ] T5b: Prove deterministic failure-path diagnostics for acquisition unavailable and readiness blocked classes.
  - Depends on: T5a
  - Paths: `scripts/**`, `docs/testing.md`
  - Verification: `bash scripts/smoke-create-site-preview.sh`, `bash scripts/smoke-site-clone-preview.sh`
- [ ] T5c: Re-run Wave 5 smokes after restart/idempotent cycle and capture evidence for `/resume-run` handoff.
  - Depends on: T5b
  - Paths: `scripts/**`, `PROGRESS.md`, `PLAN.md`
  - Verification: `bash scripts/smoke-create-site-preview.sh`, `bash scripts/smoke-site-clone-preview.sh`, `bash scripts/smoke-backup-restore.sh`

## Acceptance Mapping

- AC1 -> `/nodes` UI/API regression tests + manual route smoke -> done (T2, T4)
- AC2 -> node provision/readiness tests for local and remote sources, including provider-managed key injection -> in progress (T3a-T3e)
- AC3 -> Wave 5 smoke scripts against acquired local node target -> pending (T5a-T5c)
