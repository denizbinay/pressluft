Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md
Supersedes: none

# Spec Lifecycle

This document defines how specs are proposed, updated, and retired.

## Goals

- Keep specs authoritative and current.
- Prevent code/spec drift.
- Keep agent context small and high-signal.

## States

- `draft`: work in progress, not yet authoritative.
- `active`: approved and authoritative.
- `deprecated`: retained for history; not used for new work.

## Change Workflow

1. Propose change in a spec PR.
2. Mark affected docs in `Depends On` and `Supersedes` metadata.
3. Update `docs/spec-index.md` if document scope or category changes.
4. Update related contract files when behavior changes (`contracts/openapi.yaml`, schemas).
5. Add or update feature specs in `docs/features/` for implementation tasks.

## Review Rules

- Any change that impacts API, DB schema, or infrastructure behavior must be reviewed before implementation.
- Contradictions across active specs must be resolved in the same PR.
- If a spec and existing code conflict, spec wins until a documented exception is approved.

## Drift Management

- Weekly or per-milestone review of active specs.
- Mark stale docs by updating `Status: deprecated` and linking replacement in `Supersedes`.
- Do not keep two active specs for the same behavior surface.

## Progressive Metadata Adoption

- Metadata headers are required for all new specs.
- Existing specs adopt metadata opportunistically: if a spec is edited in a PR and lacks the header, add the header in that same PR.
- When a spec with metadata is edited, update at least `Last Reviewed` and `Depends On` as needed.
- No bulk metadata-only rewrite is required.

Current status: metadata headers are now present across all `docs/*.md` files. Continue updating `Last Reviewed` and dependencies during normal edits.

## Required PR Checklist (Specs)

- Governing spec(s) listed.
- Scope and non-scope are explicit.
- Acceptance criteria are testable.
- Contract impact declared (API/DB/infra/no-impact).
