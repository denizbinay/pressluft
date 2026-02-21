Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-lifecycle.md, docs/spec-index.md, AGENTS.md
Supersedes: none

# Changes Workflow

This document defines proposal-based workflow for major changes only.

## Purpose

- Keep high-impact changes reviewable before implementation.
- Preserve rationale and execution intent across sessions.
- Prevent undocumented architecture or contract drift.

## When Proposal Workflow Is Required

Create a `changes/<slug>/` proposal for any change that does one or more of the following:

- Changes DB schema or `migrations/`.
- Adds or removes API endpoints.
- Changes infrastructure execution model or Ansible playbook structure.
- Introduces broad multi-feature refactors.
- Changes security-critical auth/session behavior.

## When It Is Not Required

- Small implementation tasks already covered by one existing feature spec.
- Documentation-only clarifications with no contract/behavior impact.
- Test-only changes that do not alter behavior.

## Required Artifacts

Each major change folder must include:

- `proposal.md` (problem, scope, impact).
- `design.md` (technical plan, dependencies, tradeoffs).
- `tasks.md` (atomic checklist with dependencies and verification).

Optional for large changes:

- `spec-deltas/` with explicit additions/modifications/removals.

## Lifecycle

1. Draft proposal and design.
2. Review and align with canonical specs.
3. Execute tasks with verification evidence.
4. Archive by merging approved changes back into canonical docs.

## Archive Rule

After implementation completes, update canonical specs in the same PR and mark the change folder as archived in `changes/README.md`.
