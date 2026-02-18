# UI Flows

This document defines the minimal MVP UI flows that consume the API.

## Login

- User enters email and password.
- On success, session token is stored and user is redirected to Sites.

## Site Creation

- User clicks "Create Site".
- Form: name, slug.
- Submits to `POST /api/sites`.
- UI shows job progress and transitions to site view when complete.

## Environment Creation

- User selects site and clicks "Create Environment".
- Form: name, slug, type, source environment, promotion preset.
- Submits to `POST /api/sites/:id/environments`.
- UI shows job progress.

## Deployment

- User selects environment and clicks "Deploy".
- Form: source type (git or upload), source ref.
- Submits to `POST /api/environments/:id/deploy`.
- UI shows job progress and health status.

## Promotion

- User selects source environment and clicks "Promote".
- UI runs drift check and displays status.
- If drifted, show blocking warning with explicit confirmation.
- Requires fresh backup confirmation.
- Submits to `POST /api/environments/:id/promote`.

## Backups

- User clicks "Create Backup".
- Selects scope (db/files/full).
- Submits to `POST /api/environments/:id/backups`.
- Backup list shows status and retention date.

## Restore

- User selects backup and clicks "Restore".
- Confirms pre-restore backup requirement.
- Submits to `POST /api/environments/:id/restore`.

## Domains

- User adds domain hostname.
- Submits to `POST /api/environments/:id/domains`.
- UI displays TLS status.

## Jobs

- Global job list shows running and recent jobs.
- Job details show status, error messages, and timestamps.
