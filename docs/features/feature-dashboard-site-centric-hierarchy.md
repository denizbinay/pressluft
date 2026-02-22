Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/ui-flows.md, docs/technical-architecture.md, docs/features/feature-dashboard-ia-overhaul.md, docs/features/feature-site-create.md, docs/features/feature-environment-create-clone.md, docs/features/feature-backups.md, docs/features/feature-jobs-and-metrics.md
Supersedes: docs/features/feature-dashboard-ia-overhaul.md

# FEATURE: dashboard-site-centric-hierarchy

## Problem

The current dashboard hierarchy exposes top-level `environments` and `backups` surfaces that are disconnected from site context, and it does not expose first-class provider visibility. This creates operator confusion and hides critical runtime truth (provider connection status, node readiness, site-to-node placement, and deploy readiness).

## Scope

- In scope:
  - Introduce top-level dashboard subsites for `/`, `/providers`, `/nodes`, `/sites`, and `/jobs`.
  - Remove top-level `/environments` and `/backups` navigation and route serving.
  - Make environments and backups site-scoped views under `/sites`.
  - Add node readiness visibility and provider status/deploy-prepared signals.
  - Add site list columns for node assignment, status, preview URL, and WordPress version.
  - Remove seeded placeholder job/node/site/environment/backup records from the dev dashboard runtime and use real in-memory runtime state with explicit empty states.
- Out of scope:
  - Standalone Nuxt dashboard migration.
  - Provider-specific acquisition internals.
  - New deployment orchestration behavior.

## Allowed Change Paths

- `internal/devserver/**`
- `internal/api/**`
- `internal/nodes/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/ssh/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/features/feature-dashboard-site-centric-hierarchy.md`
- `docs/features/README.md`
- `PLAN.md`
- `PROGRESS.md`
- `changes/dashboard-runtime-realignment/**`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/contract-traceability.md`
- `docs/error-codes.md`

## Acceptance Criteria

1. Authenticated operators can navigate `/`, `/providers`, `/nodes`, `/sites`, and `/jobs`, and cannot navigate dedicated top-level `/environments` or `/backups` routes.
2. `/providers` shows connection status and setup guidance for provider-backed node creation.
3. `/providers` credential guidance uses bearer-token semantics and does not require static token-prefix formatting.
4. `/nodes` shows all registered nodes with status and deploy-readiness fields.
5. `/sites` shows each site with status, production preview URL, assigned node, and current WordPress version.
6. Site detail workflow in `/sites` contains the only environment and backup views, scoped to the selected site.
7. Dashboard no longer preloads mock/seed records; empty-state guidance is shown when there is no runtime data.
8. `/resume-run` picks this feature work after Wave 5.5 and before Wave 6 tasks.

## Wave 5.8 Stabilization Bug List

1. `BUG-DS-01` - `/nodes` host display falls back to `-` for provider-acquired nodes without explicit hostname/public IP fields, which hides operator-useful target context.

Wave 5.8 acceptance addendum:

1. `/nodes` host column shows deterministic provider-target fallback (`<pending-provider-target>`) when acquisition has not produced hostname/public IP yet.

## Scenarios (WHEN/THEN)

1. WHEN an authenticated operator opens `/providers` THEN provider connection status and remediation guidance are visible before node creation.
2. WHEN an operator opens `/nodes` THEN node status/readiness and provider-target host context are visible without opening any site page.
3. WHEN an operator opens `/sites` THEN site rows include node placement, status, preview URL, and WordPress version.
4. WHEN an operator needs backups or staging/clone context THEN they select a site and use site-scoped environment/backups surfaces only.
5. WHEN runtime state is empty THEN dashboard panels show explicit empty states instead of fabricated records.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `go run ./cmd/pressluft dev --port 18400` and `curl http://127.0.0.1:18400/ && curl http://127.0.0.1:18400/providers && curl http://127.0.0.1:18400/nodes && curl http://127.0.0.1:18400/sites && curl http://127.0.0.1:18400/jobs`
- Required tests:
  - `internal/devserver/server_test.go` route and marker assertions updated for site-centric hierarchy.
  - API handler tests for new node/runtime read endpoints used by dashboard.

## Risks and Rollback

- Risk: route realignment can regress existing create/list flows and deep links.
- Rollback: restore prior route map and navigation while keeping runtime worker fixes intact.
