# Tasks: wp-first-runtime

## Wave Plan

- Wave 5.5 (runtime vertical slice): self-node baseline, runnable WordPress provisioning, reachability-gated success semantics.
- Wave 6+ (dependent): deploy/update safety work resumes only after MP1.5 passes.
- Merge point(s): MP1.5 (end of Wave 5.5 before Wave 6).

## Atomic Tasks

- [x] T1: Rework plan and progress docs to insert Wave 5.5 and MP1.5 gating.
  - Depends on: none
  - Paths: `PLAN.md`, `PROGRESS.md`, `docs/plan-dependency-matrix.md`
  - Verification: `bash scripts/check-readiness.sh`
- [x] T2: Author runtime-first feature spec for self-node reachability and lifecycle semantics.
  - Depends on: T1
  - Paths: `docs/features/feature-wp-first-runtime.md`, `docs/features/README.md`
  - Verification: spec completeness review against template and governance docs
- [ ] T3: Implement self-node runtime target handling and mutation execution alignment.
  - Depends on: T2
  - Paths: `internal/jobs/**`, `internal/sites/**`, `internal/environments/**`, `internal/nodes/**`, `internal/store/**`
  - Verification: `go test ./internal/... -v`
- [ ] T4: Ensure Ansible mutation path provisions reachable WordPress runtime for site/env create.
  - Depends on: T3
  - Paths: `ansible/playbooks/**`, `ansible/roles/**`, `docs/ansible-execution.md`, `docs/provisioning-spec.md`
  - Verification: `ansible-playbook --syntax-check` for affected playbooks, runtime smoke checks
- [ ] T5: Add create-site -> preview URL reachability smoke verification and dashboard visibility.
  - Depends on: T4
  - Paths: `scripts/**`, `internal/devserver/**`, `docs/ui-flows.md`
  - Verification: scripted smoke run + `go test ./internal/... -v`

## Acceptance Mapping

- AC1 (reworked plan gate) -> `bash scripts/check-readiness.sh` + doc review -> done
- AC2 (site create yields reachable WP URL) -> Wave 5.5 manual + scripted smoke -> pending
- AC3 (success semantics require reachability) -> lifecycle tests in `internal/jobs/**` -> pending
