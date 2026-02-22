# Proposal: node-acquisition-parity

## Problem

Wave 5 runtime/provisioning behavior is reliable once nodes are ready, but local quickstart and development can fail before execution because host prerequisites are not uniformly controlled. This produces friction and weakens confidence in end-to-end parity with fresh Ubuntu nodes.

## Scope

- In scope:
  - Introduce explicit node creation semantics with two operator actions: `Create Local Node` and `Create Remote Node`.
  - Keep one canonical provisioning/readiness flow after node acquisition (SSH + Ansible + readiness model).
  - Implement local VM acquisition for quickstart/development so Wave 5 smokes run without manual runtime installs on the developer host.
  - Use `multipass` as the canonical local acquisition backend for Wave 5.11 completion.
  - Treat `multipass` as provider-equivalent lifecycle (`create/start VM -> inject Pressluft-managed SSH key -> provision`), matching future cloud-provider semantics.
  - Remove success-path dependency on provider-internal daemon key locations.
  - Prepare remote-node creation contract for future provider-backed creation without changing provisioning semantics.
- Out of scope:
  - Per-site containerized WordPress runtime.
  - Wave 6 deploy/update behavior.
  - Provider-specific remote node orchestration implementation.

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
- `docs/provisioning-spec.md`
- `docs/ansible-execution.md`
- `docs/features/feature-wave5-node-acquisition-parity.md`
- `docs/features/feature-install-bootstrap.md`

## Acceptance Criteria

1. `/nodes` creation surface exposes both `Create Local Node` and `Create Remote Node` actions, with no implicit fallback between them.
2. Local-node acquisition feeds the existing provisioning/readiness path and reaches the same terminal invariants used by remote nodes.
3. Wave 5 create-site/clone/backup-restore smokes run against an acquired local node target without requiring manual developer-host runtime setup.
4. Wave 5 success-path completion no longer depends on control-plane host-local readiness checks or seeded loopback self-node defaults.
5. Local provider success-path SSH authentication uses Pressluft-managed key generation/injection rather than provider-internal key assumptions.
