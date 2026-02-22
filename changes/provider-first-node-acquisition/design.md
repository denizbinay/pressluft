# Design: provider-first-node-acquisition

## Architecture and Boundaries

- Services/components affected:
  - `internal/providers/*` for provider connection and acquisition adapters.
  - `internal/api` for provider endpoints and provider-backed node-create contract.
  - `internal/nodes` and `internal/jobs` for acquisition handoff into `node_provision`.
  - `internal/devserver` for `/providers` and provider-first `/nodes` UX.
- Ownership boundaries:
  - Provider adapters own acquisition lifecycle.
  - Node provisioning/readiness remains provider-agnostic after acquisition output.
  - Infrastructure mutations remain job queue + Ansible only.

## Data Flow Summary

1. Operator connects provider in `/providers` (secret persisted).
2. Operator creates node in `/nodes` using provider-backed create request.
3. `POST /api/nodes` returns `202` + job id.
4. `node_provision` executes Hetzner lifecycle (`create server -> poll action -> fetch server`).
5. Acquisition output becomes SSH target for existing provisioning + readiness flow.

## Technical Plan

1. Add provider connection model + API + dashboard route (`/providers`).
2. Update node-create API contract to provider-backed request shape only.
3. Implement Hetzner adapter with `hcloud-go`, deterministic error mapping, and retries.
4. Remove local/manual/self-node acquisition runtime paths.
5. Update smoke scripts to run against provider-acquired node target and capture acquisition evidence.

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli

## Risks and Mitigations

- Risk: provider API outages or rate limiting.
  - Mitigation: classify retryable vs terminal provider errors; surface actionable status in `/providers`.
- Risk: credential leaks.
  - Mitigation: persist as secrets, redact logs, avoid echoing tokens in responses/tests.
- Risk: stale local/manual assumptions remain in tests/docs.
  - Mitigation: explicit cleanup task and grep-based verification for deprecated paths.

## Rollback

1. Disable provider-backed create mutations via feature gate.
2. Keep provider connection read-paths for diagnostics.
3. Revert to last stable provider-backed contract version while keeping local/manual create removed from active plan.
