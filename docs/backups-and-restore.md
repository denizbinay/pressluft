# Backups and Restore

This document defines backup storage, retention, and restore behavior for MVP.

## Backup Types

- Database-only backup.
- Files-only backup.
- Full backup (database + files).

## Storage

- Off-site storage is required.
- S3-compatible storage is the MVP target.
- Storage credentials are stored as secrets in the control plane.

## Retention

- Time-based retention only.
- Default retention: 30 days.
- `retention_until` is computed at backup creation time.

## Backup Workflow

1. Create a consistent database export using single-transaction mode.
2. Archive files (excluding cache directories).
3. Upload to S3-compatible storage.
4. Record checksum and size.
5. Mark status as completed.

## Restore Workflow

1. Validate backup checksum before restore.
2. Create a pre-restore full backup of the target environment.
3. Restore files and database.
4. Run health checks.
5. If health checks fail, rollback automatically.

## Restore Targets

- Any backup can be restored to production or non-production environments.
- Restore is environment-scoped and must not affect other sites.

## Cleanup

- A scheduled cleanup job removes backups where `retention_until` is in the past.
