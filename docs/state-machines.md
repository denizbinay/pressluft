# State Machines

This document defines allowed state transitions and rules for MVP resources.

## Site

States: `active | cloning | deploying | restoring | failed`

Transitions:

- `active -> cloning` when a clone/staging creation job starts.
- `active -> deploying` when a deploy/update job starts.
- `active -> restoring` when a restore job starts.
- `cloning -> active` on successful clone creation.
- `deploying -> active` on successful deploy.
- `restoring -> active` on successful restore.
- `cloning|deploying|restoring -> failed` on unrecoverable job failure.
- `failed -> active` only via explicit admin reset action after validation.

Rules:

- Only one mutation job per site may run at a time.
- Site status reflects the most impactful environment mutation currently running.

## Environment

States: `active | cloning | deploying | restoring | failed`

Transitions:

- `active -> cloning` when environment clone starts.
- `active -> deploying` when environment deployment starts.
- `active -> restoring` when environment restore starts.
- `cloning -> active` when clone creation completes successfully.
- `deploying -> active` when deploy completes successfully.
- `restoring -> active` when restore completes successfully.
- `cloning|deploying|restoring -> failed` on unrecoverable failure.
- `failed -> active` only via explicit admin reset action after validation.

Rules:

- Environment status mirrors the job executing against it.
- `current_release_id` must point to a valid release when status is `active`.

## Job

States: `queued | running | succeeded | failed | cancelled`

Transitions:

- `queued -> running` when a worker locks and starts the job.
- `running -> succeeded` on successful completion.
- `running -> failed` when max attempts exceeded or non-retryable error occurs.
- `running -> queued` on retryable failure (increment attempt_count).
- `queued|running -> cancelled` only via explicit admin cancellation.

Rules:

- Jobs are idempotent and safe to retry.
- At-least-once execution is assumed.

## Backup

States: `pending | running | completed | failed | expired`

Transitions:

- `pending -> running` when backup job starts.
- `running -> completed` when backup upload completes and checksum is recorded.
- `running -> failed` on backup failure.
- `completed -> expired` when retention_until passes.

Rules:

- Expired backups are eligible for deletion by a cleanup job.

## Release

Health states: `unknown | healthy | unhealthy`

Transitions:

- `unknown -> healthy` on successful health check.
- `unknown -> unhealthy` on failed health check.
- `healthy -> unhealthy` if subsequent checks fail.

Rules:

- Rollback is triggered automatically if the active release becomes unhealthy.
