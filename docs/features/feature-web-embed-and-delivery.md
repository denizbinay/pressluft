Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/testing.md, docs/ui-flows.md
Supersedes: none

# FEATURE: web-embed-and-delivery

## Problem

MVP architecture requires an embedded web UI in the Go control plane, but current server routes expose API/admin endpoints only.

## Scope

- In scope:
  - Serve built dashboard assets through the Go binary/control plane.
  - Preserve API and admin route behavior while adding SPA fallback for dashboard routes.
  - Define deterministic build/serve expectations for local and CI usage.
- Out of scope:
  - CDN edge caching strategy.
  - Multi-binary distribution model.

## Allowed Change Paths

- `cmd/pressluft/**`
- `internal/api/**`
- `internal/**`
- `web/**`
- `.github/workflows/ci.yml`
- `docs/features/feature-web-embed-and-delivery.md`
- `docs/testing.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. Built dashboard is reachable from the control-plane HTTP server without a separate Node process in production mode.
2. Client-side dashboard routes resolve via SPA fallback while `/api/*` and `/_admin/*` keep existing behavior.
3. Startup/packaging docs and CI commands validate deterministic dashboard delivery.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
- Required tests:
  - HTTP handler tests for static asset serving and SPA fallback precedence.

## Risks and Rollback

- Risk: route precedence regressions can break API handlers.
- Rollback: keep explicit route separation and fallback only on non-API/non-admin paths.
