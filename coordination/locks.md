# Parallel Lock Registry

Use this file to claim and release file ownership during parallel execution.

## Active Locks

| lock_id | owner | paths | claimed_at_utc | expected_minutes | status | note |
|---|---|---|---|---:|---|---|
| wave6/closeout/opencode-ses_37e7db | opencode | `PLAN.md`, `PROGRESS.md`, `coordination/locks.md` | 2026-02-21T19:06:57Z | 30 | released | Mark W6-T2/W6-T3 complete; advance next task. |
| wave7/w7-t1/opencode-ses_37e7db | opencode | `web/**`, `docs/features/feature-web-sites-and-environments.md`, `coordination/locks.md` | 2026-02-21T19:07:58Z | 120 | released | Implement sites/environments list/detail/create flows and tests. |
| wave7/w7-t2/opencode-ses_37e7db | opencode | `web/**`, `docs/features/feature-web-lifecycle-workflows.md`, `coordination/locks.md` | 2026-02-21T19:17:50Z | 180 | released | Implement deploy/update/restore/promote UI flows with job progress. |
| wave7/w7-t3/opencode-ses_37e7db | opencode | `web/**`, `docs/features/feature-web-operations-workflows.md`, `coordination/locks.md` | 2026-02-21T19:26:46Z | 240 | released | Implement backups/domains/cache/magic-login workflows and tests. |
| wave7/w7-t4/opencode-ses_37e7db | opencode | `web/**`, `docs/features/feature-web-jobs-metrics-controls.md`, `coordination/locks.md` | 2026-02-21T19:32:56Z | 240 | released | Implement jobs/metrics visibility and admin control UI. |
| wave8/w8-t1/opencode-ses_37e7db | opencode | `.github/workflows/ci.yml`, `docs/features/feature-web-dashboard-hardening.md`, `coordination/locks.md` | 2026-02-21T19:39:02Z | 120 | released | Confirm frontend CI gates and command presets for verification. |
| wave8/w8-t2/opencode-ses_37e7db | opencode | `web/**`, `docs/features/feature-web-dashboard-hardening.md`, `coordination/locks.md` | 2026-02-21T19:39:45Z | 240 | released | Harden dashboard async/error/accessibility + focus-visible baseline. |
| wave8/w8-t3/opencode-ses_37e7db | opencode | `web/**`, `docs/testing.md`, `PLAN.md`, `PROGRESS.md`, `coordination/locks.md` | 2026-02-21T20:00:08Z | 120 | released | Static build now produces `index.html`; embedded dashboard smoke verified; updated docs for flag ordering. |

## Reclaimed Locks

| lock_id | previous_owner | reclaimed_by | reclaimed_at_utc | reason |
|---|---|---|---|---|
