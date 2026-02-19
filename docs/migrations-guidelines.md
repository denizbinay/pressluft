Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/data-model.md
Supersedes: none

# Migrations Guidelines

This document defines migration authoring rules for SQLite in Pressluft.

## Rules

- Migration filenames must be sequential and timestamped.
- Prefer reversible migrations; if irreversible, document why.
- No data-dependent DDL with generated IDs.
- Do not modify applied migrations; create new forward migrations.

## SQLite-Specific Practices

- Assume SQLite is canonical (no Postgres-specific SQL).
- Keep schema changes explicit and additive when possible.
- Use transaction-safe migration steps.

## Required Process

1. Update relevant spec (`docs/data-model.md` and feature spec).
2. Add migration file under `migrations/`.
3. Run migration command: `go run ./migrations/migrate.go up`.
4. Validate affected API/behavior tests.

## Readiness Note

- The repository must contain a baseline `migrations/` directory before implementation starts.
