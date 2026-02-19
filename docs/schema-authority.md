Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/data-model.md, docs/migrations-guidelines.md, docs/contract-guardrails.md
Supersedes: none

# Schema Authority

This document defines the source-of-truth chain for persistent schema and shared API shapes.

## Authority Chain

1. Conceptual model: `docs/data-model.md`
2. Executable schema changes: `migrations/` (timestamped migration files)
3. API payload contract: `contracts/openapi.yaml`

`docs/data-model.md` describes intent. `migrations/` is the executable truth for DB structure.

## Rules

- No schema mutation is complete without a migration file.
- Do not edit previously applied migrations; create a new forward migration.
- Every schema-affecting change must update both `docs/data-model.md` and impacted API contracts.
- No inline DDL in handlers/services.

## Pre-Implementation Requirement

Before implementation begins, the repository must include a `migrations/` directory and baseline initial migration matching `docs/data-model.md`.

## Verification

- `go run ./migrations/migrate.go up` succeeds on a clean DB.
- Schema and contract checks in CI confirm no enum/state drift.
