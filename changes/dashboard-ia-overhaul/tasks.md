Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/features/feature-dashboard-ia-overhaul.md, PLAN.md, PROGRESS.md
Supersedes: none

# Tasks: dashboard-ia-overhaul

## Wave Plan

- Wave 0 (foundation): W5-T3 spec and change packet finalization.
- Wave 1 (dependent): W5-T4 through W5-T7 dashboard refactor and migration.
- Merge point(s): before Wave 6 start (`W6-T1`) with backend gates passing.

## Atomic Tasks

- [x] T1: Finalize feature spec and planning docs for dashboard IA overhaul.
  - Depends on: Wave 5 backup UI baseline (W5-T2)
  - Paths: `docs/features/feature-dashboard-ia-overhaul.md`, `PLAN.md`, `PROGRESS.md`, `changes/dashboard-ia-overhaul/**`
  - Verification: `bash scripts/check-readiness.sh`
- [x] T2: Implement route-level dashboard shell and concern-based navigation.
  - Depends on: T1
  - Paths: `internal/devserver/**`
  - Verification: `go test ./internal/... -v`
- [x] T3: Refactor shared state and migrate existing views into subsites.
  - Depends on: T2
  - Paths: `internal/devserver/**`
  - Verification: `go test ./internal/... -v`
- [x] T4: Add route/marker regression tests and complete gate verification.
  - Depends on: T3
  - Paths: `internal/devserver/server_test.go`
  - Verification: `go build -o ./bin/pressluft ./cmd/pressluft`, `go vet ./...`, `go test ./internal/... -v`

## Acceptance Mapping

- AC1 -> T2 + T4 route smoke/tests -> complete
- AC2 -> T3 + T4 flow regression/tests -> complete
- AC3 -> T3 migration checks with contract-aligned errors -> complete
- AC4 -> T2/T3 module split in `internal/devserver/**` -> complete
- AC5 -> T1 updates in `PLAN.md` and `PROGRESS.md` -> complete
