# Design: wp-first-runtime

## Architecture and Boundaries

- Services/components affected:
  - Job orchestration and lifecycle handling (`internal/jobs/**`).
  - Site/environment mutation services (`internal/sites/**`, `internal/environments/**`).
  - Node target semantics for local/self-node (`internal/nodes/**`, `internal/store/**`).
  - Ansible execution surfaces for mutation runtime provisioning (`ansible/playbooks/**`, `ansible/roles/**`).
- Ownership boundaries:
  - Control plane (Go) keeps transactional state and queue authority.
  - All node mutations remain via queue + Ansible.
  - Node queries for validation remain direct SSH per architecture constraints.
- Data flow summary:
  1. Operator triggers site/environment create.
  2. API enqueues mutation job.
  3. Worker executes Ansible runtime provisioning on selected node target.
  4. Reachability validation runs before final success transition.
  5. Job/environment/site state commits transactionally.

## Technical Plan

1. Add Wave 5.5 gate and dependencies in planning docs (`PLAN.md`, `PROGRESS.md`, `docs/plan-dependency-matrix.md`).
2. Define self-node local/WSL2 execution constraints and expected node model behavior in runtime feature/spec docs.
3. Align mutation execution contract so `site_create`/`env_create` only succeed after runtime reachability check.
4. Add deterministic smoke path and operator-visible failure diagnostics.
5. Keep Wave 6+ unchanged in intent, but dependent on MP1.5 completion.

## Dependencies

- Depends on:
  - Completed Wave 5 dashboard and queue scaffolding.
  - Existing node provisioning and mutation queue invariants.
- Blocks:
  - Wave 6 deploy/update safety automation.

## Risks and Mitigations

- Risk: self-node behavior diverges between host Linux and WSL2.
  - Mitigation: define WSL2 baseline as canonical acceptance target and document network assumptions.
- Risk: partial runtime success can still appear healthy in state transitions.
  - Mitigation: enforce explicit runtime reachability check before success commit.
- Risk: over-expanding scope into unrelated lifecycle features.
  - Mitigation: constrain Wave 5.5 to site/environment creation runtime viability only.

## Rollback

- Safe rollback sequence:
  1. Revert Wave 5.5 runtime-specific lifecycle gating changes.
  2. Restore prior mutation success criteria while keeping queue invariants and audit behavior intact.
  3. Keep planning artifacts, mark packet as superseded, and re-open scope with narrowed acceptance.
