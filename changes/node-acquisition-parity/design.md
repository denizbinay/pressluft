# Design: node-acquisition-parity

## Architecture and Boundaries

- Services/components affected:
  - `internal/nodes` for acquisition source handling and readiness handoff.
  - `internal/devserver` for `/nodes` create controls and operator messaging.
  - `internal/api` for node-create contract updates.
  - `internal/bootstrap` for quickstart/install-driven local node acquisition hooks.
  - `internal/jobs` and existing playbooks for unchanged provisioning execution semantics.
- Ownership boundaries:
  - Node acquisition is the only source-specific layer.
  - Provisioning, readiness checks, and mutation execution remain source-agnostic.
  - Infrastructure mutations continue through job queue + Ansible.
- Data flow summary:
  1. Operator chooses `Create Local Node` or `Create Remote Node`.
  2. Selected acquisition path completes provider lifecycle and produces a reachable SSH target.
  3. Provider lifecycle for local path includes Pressluft-managed SSH key generation/reuse and key injection into VM user `authorized_keys`.
  4. Existing node-provision job runs against target.
  5. Existing readiness model determines node usability for create flows.

## Implementation Decision (Wave 5.11)

- Canonical local VM backend: `multipass`.
- If `multipass` capability is unavailable, local creation fails deterministically with explicit remediation; no source fallback is allowed.

## Technical Plan

1. Add node acquisition contract with explicit source type and no-fallback enforcement.
2. Add `/nodes` create UI actions for local and remote creation paths.
3. Implement local VM acquisition adapter for quickstart/development using `multipass`.
4. Add provider-equivalent local lifecycle (`create/start -> inject key -> return target`) and remove reliance on provider-internal key paths.
5. Remove success-path dependency on auto-seeded loopback self-node records.
6. Route both acquisition results into existing node provisioning and SSH-targeted readiness checks.
7. Extend smoke/regression scripts and docs to use acquired local node path for Wave 5 verification.

## No-Fallback Matrix

- Local selected + acquisition failed -> return local capability/acquisition error; do not create remote node.
- Local selected + SSH key injection failed -> return deterministic local acquisition/provisioning failure; do not continue with provider-internal key assumptions.
- Local selected + provisioning failed -> return local provisioning failure; do not create remote node.
- Remote selected + validation/acquisition failed -> return remote path error; do not create local node.
- Remote selected + provisioning failed -> return remote provisioning failure only.

## Readiness Execution Path

- After acquisition, readiness checks execute against the acquired node target over SSH for both local and remote sources.
- `sudo_unavailable` and `runtime_missing` represent acquired-node failures, not control-plane host shell state.

## Dependencies

- Depends on:
  - Wave 5.10 completion for backup/restore e2e baseline.
  - Existing readiness reason-code model and node provisioning playbook.
- Blocks:
  - Wave 6 start until Wave 5 node acquisition parity is verified.

## Risks and Mitigations

- Risk: local VM tooling support varies by host environment.
  - Mitigation: keep provider adapter isolated and return deterministic capability errors.
- Risk: acquisition errors might silently fall through to alternate path.
  - Mitigation: enforce source-typed request validation and explicit no-fallback test coverage.
- Risk: provider key assumptions differ by host packaging (for example snap-managed daemon paths).
  - Mitigation: use Pressluft-managed key lifecycle and provider-side key injection; never read provider-internal daemon private keys on success path.
- Risk: parity effort regresses current `/nodes` and `/sites` operator flows.
  - Mitigation: keep existing route and readiness tests as required regression gates.

## Rollback

- Safe rollback sequence:
  1. Disable local-node acquisition adapter behind an explicit feature flag/capability gate.
  2. Keep remote-node and existing provisioning/readiness behavior unchanged.
  3. Revert node-creation contract changes with matching docs/OpenAPI rollback.
