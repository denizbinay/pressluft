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

## Site Import

- User selects site and clicks "Import".
- Form: archive URL.
- Submits to `POST /api/sites/:id/import`.
- UI shows job progress.

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
- UI runs drift check and displays status via `POST /api/environments/:id/drift-check`.
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

## Caching

The environment detail view includes a Caching section with per-environment controls.

- Two toggles are displayed: **FastCGI Page Cache** and **Redis Object Cache**. Each shows the current state (enabled/disabled) read from the environment record.
- Toggling either submits to `PATCH /api/environments/:id/cache` with the changed value. UI shows job progress while the Nginx server block and/or Redis drop-in are reconfigured.
- A **"Purge Cache"** button submits to `POST /api/environments/:id/cache/purge`. UI shows job progress.
- Both caches are enabled by default for new environments.

## Magic Login

- The environment detail view includes an **"Open WordPress Admin"** button.
- Clicking it submits to `POST /api/environments/:id/magic-login`.
- On success (response contains `login_url`), the URL is opened in a new browser tab.
- On failure, an inline error message is displayed (e.g., "Could not connect to server" for `node_unreachable`, or "Environment is not active" for `environment_not_active`).
- The button is only enabled when the environment status is `active`.
- No job progress indicator is shown — this is a synchronous operation that completes in approximately 1 second.

## Domains

- User adds domain hostname.
- Submits to `POST /api/environments/:id/domains`.
- UI displays TLS status.

- User removes a domain.
- Submits to `DELETE /api/domains/:id`.

## Updates

- User selects environment and clicks "Apply Updates".
- Selects scope (core/plugins/themes/all).
- Submits to `POST /api/environments/:id/updates`.

## Jobs

- Global job list shows running and recent jobs.
- Job details show status, error messages, and timestamps via `GET /api/jobs` and `GET /api/jobs/:id`.
