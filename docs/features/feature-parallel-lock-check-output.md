Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/parallel-execution.md
Supersedes: none

# FEATURE: parallel-lock-check-output

## Problem

`bash scripts/check-readiness.sh` runs the parallel lock validator, but its success message currently reports the number of lock table rows as "active lock rows".

This is misleading because the `## Active Locks` table may include rows with `status=released`.

## Scope

- In scope:
  - Make the parallel lock validator success output accurately report active vs total lock rows.
- Out of scope:
  - Changing the lock table format.
  - Changing lock lifecycle policy (how/when rows are moved between sections).

## Allowed Change Paths

- `scripts/check-parallel-locks.sh`
- `docs/features/feature-parallel-lock-check-output.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. `bash scripts/check-parallel-locks.sh` prints a non-misleading success message that distinguishes active locks from total lock table rows.
2. `bash scripts/check-readiness.sh` continues to pass on a healthy repo.

## Verification

- Required commands:
  - `bash scripts/check-parallel-locks.sh`
  - `bash scripts/check-readiness.sh`
