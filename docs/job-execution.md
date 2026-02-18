# Job Execution

This document defines the MVP job execution model for the control plane.

## Guarantees

- At-least-once execution. Jobs must be idempotent.
- Only one job per site at a time.
- Only one job per node at a time.
- Jobs that touch a node must include `node_id`.

## Queueing

- Jobs are stored in the `jobs` table.
- Workers poll for `status = queued` and `run_after <= now`.
- Default polling interval: 2 seconds.

## Locking

- When a worker starts a job, it sets:
  - `status = running`
  - `locked_at = now`
  - `locked_by = <worker_id>`
  - `started_at = now`
- Lock acquisition is atomic and includes a site/node concurrency check.
- Concurrency check uses a transaction to ensure no other running job exists for the same site or node.

## Timeouts

- Default job timeout: 30 minutes.
- If a job exceeds timeout, it is marked failed with `error_code = JOB_TIMEOUT`.

## Retries

- Default max attempts: 3.
- Retryable errors set `status = queued` and `run_after = now + backoff`.
- Backoff schedule: 1 min, 5 min, 15 min.
- Non-retryable errors set `status = failed`.

## Error Handling

- All node-targeted jobs are executed via Ansible (see `docs/ansible-execution.md`).
- Ansible exit codes are mapped to retryable and non-retryable errors as defined in the Ansible execution spec.
- Errors must set `error_code` and `error_message`.
- If max attempts exceeded, set `status = failed`.

## State Synchronization

- Job start transitions the related site/environment to the corresponding in-progress state (`cloning`, `deploying`, or `restoring`).
- Job completion transitions site/environment back to `active` or `failed`.
- These transitions occur inside the same DB transaction as job state updates.

## Cancellation

- Only admin can cancel.
- Cancellation sets `status = cancelled` and does not attempt rollback.
- If cancellation happens while running, the job handler must perform safe stop where possible.

## Worker Identity

- `locked_by` is a stable identifier for the worker process (hostname + pid).
