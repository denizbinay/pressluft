# Tasks: dashboard-runtime-realignment

## Wave Plan

- Wave 0 (foundation): finish Wave 5.5 smoke unblock and gate checks.
- Wave 1 (dependent): implement site-centric dashboard hierarchy and runtime inventory endpoints.
- Merge point(s): MP1.5 (Wave 5.5 + Wave 5.6 complete before Wave 6).

## Atomic Tasks

- [ ] T1: Complete Wave 5.5 smoke unblock by wiring dev runtime worker loop and validating create-site preview reachability.
  - Depends on: `changes/wp-first-runtime/tasks.md` T4
  - Paths: `internal/devserver/**`, `internal/jobs/**`, `cmd/pressluft/**`, `scripts/smoke-create-site-preview.sh`, `PLAN.md`, `PROGRESS.md`
  - Verification: `bash scripts/smoke-create-site-preview.sh`, `go test ./internal/... -v`
- [x] T2: Author and align feature specs for dashboard hierarchy realignment and runtime inventory queries.
  - Depends on: none
  - Paths: `docs/features/feature-dashboard-site-centric-hierarchy.md`, `docs/features/feature-runtime-inventory-queries.md`, `docs/features/README.md`, `PLAN.md`, `PROGRESS.md`
  - Verification: `bash scripts/check-readiness.sh`
- [ ] T3: Implement and document nodes/runtime-inventory API contract additions.
  - Depends on: T1, T2
  - Paths: `contracts/openapi.yaml`, `docs/api-contract.md`, `docs/contract-traceability.md`, `docs/error-codes.md`, `internal/api/**`, `internal/ssh/**`, `internal/nodes/**`, `internal/environments/**`
  - Verification: `go test ./internal/... -v`
- [ ] T4: Implement dashboard route hierarchy realignment and remove seeded placeholder data.
  - Depends on: T3
  - Paths: `internal/devserver/**`, `docs/ui-flows.md`
  - Verification: `go run ./cmd/pressluft dev --port 18400` and `curl http://127.0.0.1:18400/ && curl http://127.0.0.1:18400/nodes && curl http://127.0.0.1:18400/sites && curl http://127.0.0.1:18400/jobs`
- [ ] T5: Finalize verification gates and unblock Wave 6.
  - Depends on: T4
  - Paths: `PLAN.md`, `PROGRESS.md`, `docs/plan-dependency-matrix.md`
  - Verification: `bash scripts/check-readiness.sh`, `go build -o ./bin/pressluft ./cmd/pressluft`, `go vet ./...`, `go test ./internal/... -v`

## Acceptance Mapping

- AC1 -> `bash scripts/smoke-create-site-preview.sh` + `go test ./internal/... -v` -> pending
- AC2 -> route smoke and `internal/devserver` tests -> pending
- AC3 -> API handler tests and dashboard rendering checks -> pending
- AC4 -> contract/doc sync checks (`check-readiness`) -> pending
