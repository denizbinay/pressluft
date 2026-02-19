Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/domain-and-routing.md, docs/provisioning-spec.md, docs/api-contract.md, docs/ui-flows.md, contracts/openapi.yaml
Supersedes: none

# FEATURE: domains-and-tls

## Problem

Operators need reliable custom domain lifecycle with DNS validation and TLS automation.

## Scope

- In scope:
  - Implement `GET /api/environments/:id/domains`, `POST /api/environments/:id/domains`, and `DELETE /api/domains/:id`.
  - Enqueue `domain_add` and `domain_remove` jobs.
  - Enforce DNS-to-node validation and TLS status updates.
- Out of scope:
  - DNS record automation APIs.
  - Non-LetsEncrypt issuers for MVP.

## Allowed Change Paths

- `internal/api/**`
- `internal/domains/**`
- `internal/jobs/**`
- `internal/store/**`
- `ansible/playbooks/domain-add.yml`
- `ansible/playbooks/domain-remove.yml`
- `contracts/openapi.yaml`
- `docs/api-contract.md`
- `docs/features/feature-domains-and-tls.md`

## Contract Impact

- API: `update-required`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `contracts/openapi.yaml`
- `docs/api-contract.md`

## Acceptance Criteria

1. Add domain endpoint returns `202` with `job_id` and creates pending domain record.
2. Remove domain endpoint returns `202` with `job_id` and removes routing/cert on completion.
3. Domain list endpoint returns authoritative domain records with TLS state.
4. DNS mismatch yields deterministic failure message and job failure code.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/domain-add.yml`
  - `ansible-playbook --syntax-check ansible/playbooks/domain-remove.yml`
- Required tests:
  - Domain add/remove/list handler tests.
  - DNS verification error-path tests.

## Risks and Rollback

- Risk: incorrect primary domain update can break WordPress URLs.
- Rollback: restore prior primary domain and rerun URL rewrite safely.
