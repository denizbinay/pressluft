Status: active
Owner: platform
Last Reviewed: 2026-02-22
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/provisioning-spec.md, docs/security-and-secrets.md, docs/api-contract.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: wave5-runtime-readiness

## Problem

Wave 5 create flows currently enqueue work that cannot succeed when node prerequisites are missing (SSH daemon, auth path, sudo/become, runtime dependencies), causing late failures and poor operator guidance.

## Scope

- In scope:
  - Define a runtime readiness model for provider-acquired nodes after acquisition.
  - Add deterministic preflight checks used by `site_create` and `env_create` flows.
  - Surface stable readiness reason codes and actionable remediation guidance in API and dashboard.
  - Ensure readiness failures occur before non-viable mutation execution.
  - Ensure provider acquisition-in-progress is represented as a non-ready state until async acquisition/provisioning reaches active readiness.
  - Enforce provider parity where all acquired-node readiness probes use SSH-targeted execution semantics.
- Out of scope:
  - New node provisioning APIs.
  - Multi-node scheduler redesign.
  - Wave 6 deploy/update behavior.
  - Control-plane host shell checks as the success-path readiness source for local acquisition.

## Allowed Change Paths

- `internal/nodes/**`
- `internal/sites/**`
- `internal/environments/**`
- `internal/api/**`
- `internal/devserver/**`
- `internal/ssh/**`
- `internal/audit/**`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/error-codes.md`
- `docs/ui-flows.md`
- `docs/features/feature-wave5-runtime-readiness.md`
- `docs/features/feature-runtime-inventory-queries.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `none`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/error-codes.md`

## Acceptance Criteria

1. Node readiness is computed from explicit prerequisites and returned with stable reason codes.
2. `POST /api/sites` and `POST /api/sites/{id}/environments` reject non-ready targets before enqueueing non-viable jobs.
3. Readiness failures are operator-visible in `/nodes` and site create flows with actionable guidance.
4. Existing Wave 5 route hierarchy and site-detail scoping remain unchanged.
5. Readiness and preflight behavior are covered by unit/API tests.
6. Provider-acquired nodes use parity probing semantics over SSH; reason codes map to node-target failures rather than control-plane host state.
7. Node create requests remain async (`202`) while readiness reflects background acquisition/provisioning progress.

## Scenarios (WHEN/THEN)

1. WHEN a provider-acquired node lacks SSH/sudo/runtime prerequisites THEN create flows fail fast with stable readiness codes and no false-positive job success.
2. WHEN node prerequisites are satisfied THEN site and environment create flows enqueue and execute normally.
3. WHEN `/nodes` is viewed THEN readiness state and remediation guidance are visible without opening site detail.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
- Required tests:
  - Readiness computation tests (`internal/nodes/**`).
  - API preflight failure/success tests for site/environment create.
  - Dashboard marker/assertion updates for readiness messaging.

## Risks and Rollback

- Risk: overly strict readiness checks can block valid nodes.
- Rollback: retain reason-code logging and temporarily downgrade checks to warning-only while preserving preflight telemetry.
