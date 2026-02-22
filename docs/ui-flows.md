Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/api-contract.md
Supersedes: none

# UI Flows

This document defines the minimal MVP UI flows that consume the API.

## Login

- User enters email and password.
- On success, session token is stored and user is redirected to Sites.

## Dashboard Navigation (Wave 5.11)

- Top-level operator navigation is `/`, `/providers`, `/nodes`, `/sites`, and `/jobs`.
- There is no dedicated top-level `/environments` or `/backups` dashboard route.
- `/providers` is the provider connection and status surface.
- `/nodes` lists all nodes and highlights local-node readiness.
- `/nodes` shows readiness reason codes and remediation guidance for non-ready nodes.
- `/nodes` create view exposes one provider-backed action: `Create Node`.
- Node creation targets a connected provider and never switches to local/manual fallback.
- `/sites` is the site index/create surface.
- `/sites/{site_id}` is the dedicated site details surface for environment and backup management.
- Site rows expose a three-point quick actions menu for site-detail and site-scoped actions.

## Provider Connections

- Operator opens `/providers` and submits provider credentials.
- `POST /api/providers` persists secret material and returns provider connection state.
- UI never displays raw secret values; it only shows whether a secret is configured.
- Provider status (`connected`, `degraded`, `disconnected`) and guidance are visible before node workflows.

## Node Creation

- Operator opens `/nodes` and submits `Create Node` for a connected provider.
- Creation is provider-backed only; local/manual SSH creation is not available.
- Create is async: UI receives accepted job response, then shows queued/running/retrying/terminal outcome from jobs timeline.
- Transient provider preparation states are handled in background retries and do not require repeated button clicks.

## Site Creation

- User clicks "Create Site".
- Form: name, slug.
- Submits to `POST /api/sites`.
- If readiness preflight fails, UI shows `409 node_not_ready` guidance and no job is enqueued.
- UI shows job progress and transitions to site view when complete.

### Wave 5.5 Smoke (Create Site -> Preview Reachability)

- `scripts/smoke-create-site-preview.sh` is retired and no longer part of required unattended verification.

### Wave 5.9 Smoke (Create Clone -> Preview Reachability)

- Run `scripts/smoke-site-clone-preview.sh` from repo root.
- Script starts `pressluft dev`, authenticates, creates site + clone environment, and polls both jobs until terminal state.
- Smoke passes only when clone preview URL returns HTTP `200`, `301`, or `302`.

### Wave 5.10 Smoke (Backup + Restore)

- Run `scripts/smoke-backup-restore.sh` from repo root.
- Script starts `pressluft dev`, creates site, creates full backup, validates checksum/size metadata, and submits restore for selected environment.
- Smoke passes only when restore job succeeds and post-restore preview URL returns HTTP `200`, `301`, or `302`.

## Site Import

- User selects site and clicks "Import".
- Form: archive URL.
- Submits to `POST /api/sites/{id}/import`.
- UI shows job progress.

## Environment Creation

- User opens a site detail at `/sites/{site_id}` and clicks "Create Environment".
- Form: name, slug, type, source environment, promotion preset.
- Submits to `POST /api/sites/{id}/environments`.
- If readiness preflight fails, UI shows `409 node_not_ready` guidance and no job is enqueued.
- UI shows job progress.

## Promotion

- User selects source environment and clicks "Promote".
- UI runs drift check and displays status via `POST /api/environments/{id}/drift-check`.
- If drifted, show blocking warning and keep promote action disabled (no override path).
- Requires fresh backup confirmation.
- Submits to `POST /api/environments/{id}/promote`.

## Backups

- User opens a site detail at `/sites/{site_id}` and clicks "Create Backup".
- Selects scope (db/files/full).
- Submits to `POST /api/environments/{id}/backups`.
- Backup list shows status and retention date.

## Restore

- User selects backup and clicks "Restore".
- Confirms pre-restore backup requirement.
- Submits to `POST /api/environments/{id}/restore`.

## Caching

The environment detail view includes a Caching section with per-environment controls.

- Two toggles are displayed: **FastCGI Page Cache** and **Redis Object Cache**. Each shows the current state (enabled/disabled) read from the environment record.
- Toggling either submits to `PATCH /api/environments/{id}/cache` with the changed value. UI shows job progress while the Nginx server block and/or Redis drop-in are reconfigured.
- A **"Purge Cache"** button submits to `POST /api/environments/{id}/cache/purge`. UI shows job progress.
- Both caches are enabled by default for new environments.

## Magic Login

- The environment detail view includes an **"Open WordPress Admin"** button.
- Clicking it submits to `POST /api/environments/{id}/magic-login`.
- On success (response contains `login_url`), the URL is opened in a new browser tab.
- On failure, an inline error message is displayed (e.g., "Could not connect to server" for `node_unreachable`, or "Environment is not active" for `environment_not_active`).
- The button is only enabled when the environment status is `active`.
- No job progress indicator is shown — this is a synchronous operation that completes in approximately 1 second.

## Domains

- User adds domain hostname.
- Submits to `POST /api/environments/{id}/domains`.
- UI displays TLS status.

- User removes a domain.
- Submits to `DELETE /api/domains/{id}`.

## Jobs

- Global job list shows running and recent jobs.
- Job details show status, error messages, and timestamps via `GET /api/jobs` and `GET /api/jobs/{id}`.

## Job Control

- Operators can cancel queued or running jobs from job details via `POST /api/jobs/{id}/cancel`.
- Operators can reset failed sites and environments via `POST /api/sites/{id}/reset` and `POST /api/environments/{id}/reset` after validation.
