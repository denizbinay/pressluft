Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/spec-index.md, docs/domain-and-routing.md, docs/config-matrix.md, docs/security-and-secrets.md
Supersedes: none

# FEATURE: settings-domain-config

## Problem

Operators need a deterministic way to manage platform-level domain and DNS-01 settings that drive preview URL and TLS behavior.

## Scope

- In scope:
  - Define canonical control-plane configuration behavior for managing `control_plane_domain`, `preview_domain`, `dns01_provider`, and `dns01_credentials_json`.
  - Define an authenticated internal admin settings API under `/_admin/settings/*` for deterministic read/update flows.
  - Enforce cross-field validation and secret-storage rules.
  - Ensure settings changes update downstream provisioning/runtime behavior predictably.
- Out of scope:
  - Public `/api/settings` endpoints in MVP OpenAPI.
  - DNS record automation at provider APIs.
  - Non-ACME TLS issuer configuration.

## Allowed Change Paths

- `internal/api/**`
- `internal/settings/**`
- `internal/store/**`
- `internal/secrets/**`
- `docs/domain-and-routing.md`
- `docs/config-matrix.md`
- `docs/features/feature-settings-domain-config.md`

## Contract Impact

- API: `internal-only update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `docs/domain-and-routing.md`
- `docs/config-matrix.md`

Internal contract note:

- Settings flows are served by internal admin endpoints (`/_admin/settings/*`) and are intentionally excluded from public OpenAPI.

## Acceptance Criteria

1. Settings surface supports deterministic read/update flows for all required domain/TLS configuration keys.
2. Validation blocks invalid combinations (for example, `preview_domain` without DNS-01 provider credentials).
3. DNS credentials are persisted only through encrypted secret storage paths.
4. Settings updates produce deterministic effects on newly created environments and node provisioning inputs.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Settings validation tests for all required key combinations.
  - Settings handler/service tests for read/update behavior and redaction of secret values.

## Risks and Rollback

- Risk: invalid settings writes can break TLS issuance and preview URL routing.
- Rollback: restore last known-good settings snapshot and re-run provisioning reconciliation.
