# Proposal: provider-first-node-acquisition

## Problem

Wave 5 node creation still includes local and manual remote SSH paths that conflict with provider-backed operational goals and make closeout verification non-deterministic.

## Scope

- In scope:
  - Replace Wave 5 node acquisition with provider-backed semantics only.
  - Add provider connection surface (`/providers`) with persisted secret handling.
  - Implement Hetzner-first asynchronous acquisition inside `node_provision` via `hcloud-go`.
  - Replace token-prefix assumptions with live bearer-token validation in provider connection health.
  - Remove local/manual/self-node acquisition behavior and stale docs/tests/scripts.
- Out of scope:
  - Additional provider implementations beyond Hetzner.
  - Wave 6 deployment/update/rollback behavior.

## Impact Summary

- API contract: update-required
- DB schema/migrations: none
- Infra/playbooks: update-required
- Security posture: update-required

## Governing Specs

- `docs/spec-index.md`
- `docs/technical-architecture.md`
- `docs/job-execution.md`
- `docs/security-and-secrets.md`
- `docs/ansible-execution.md`
- `docs/features/feature-wave5-provider-first-node-acquisition.md`
- `docs/features/feature-provider-connections.md`
- `docs/features/feature-hetzner-node-provider.md`
- `docs/features/feature-hetzner-sdk-integration.md`

## Acceptance Criteria

1. Node creation is provider-backed only and remains async (`202` + job lifecycle).
2. `/providers` route provides connection and health visibility with persisted provider secrets.
3. Hetzner create/poll/fetch lifecycle feeds existing provisioning/readiness pipeline.
4. Local/manual/self-node acquisition artifacts are removed from active Wave 5 scope and runtime.
5. Wave 5 smokes pass on provider-acquired nodes with deterministic evidence.

## External References

- Hetzner Cloud Go SDK: https://github.com/hetznercloud/hcloud-go
- Hetzner Cloud CLI (background only, not runtime dependency): https://github.com/hetznercloud/cli
