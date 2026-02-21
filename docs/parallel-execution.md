Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: AGENTS.md, docs/agent-governance.md, docs/agent-session-playbook.md
Supersedes: none

# Parallel Execution

This document defines how to run multiple agents safely on the same repository.

## Execution Model

- Use one Git worktree per parallel agent for implementation work.
- Keep one owning feature spec per agent session.
- Use wave-based scheduling from `PLAN.md` to coordinate dependency order.

## File Ownership

- Before edits, claim file ownership in `coordination/locks.md`.
- Do not edit files already owned by another active agent.
- If a shared file is required, schedule a merge point and handoff.

## Lock Convention

- Lock identifier format: `<wave>/<task>/<agent-id>`.
- Lock record must include: owner, claimed paths, start time, intended duration.
- Lock timeout is 2 hours; expired locks are reclaimable.
- Lock records must use UTC ISO-8601 timestamps (`YYYY-MM-DDTHH:MM:SSZ`).
- Validate lock records with `bash scripts/check-parallel-locks.sh`.

## Stale Lock Recovery

1. Verify lock age exceeds timeout.
2. Record reclaim decision and reason.
3. Reassign the task with updated ownership.
4. Re-run relevant verification for reclaimed paths.

## Lock Registry Format

- File: `coordination/locks.md`
- Active locks belong under `## Active Locks`.
- Each active lock row must include:
  - `lock_id`
  - `owner`
  - `paths`
  - `claimed_at_utc`
  - `expected_minutes`
  - `status`
  - `note`

## Merge Points

- Define merge points in `PLAN.md` where parallel waves converge.
- At merge points, run contract and verification checks before advancing.
- Reject merge point completion if checks fail.

## Conflict Resolution

1. Prefer ownership transfer over simultaneous editing.
2. If transfer is not possible, split scope into smaller tasks.
3. If still conflicting, escalate to architect/reviewer role for decision.
