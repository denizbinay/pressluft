Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/data-model.md
Supersedes: none

# Migration Specification

This document defines the MVP migration/import contract.

## Scope

- Import a single WordPress site into a new environment via `archive_url`.
- Perform serialized-safe URL rewrite.
- Minimize downtime by validating before cutover.

## Archive Format

The import archive is a `.tar.gz` file containing:

- `db.sql` (required): database export.
- `files/` (required): WordPress `wp-content` directory contents.
- `meta.json` (optional): metadata including `source_url`.

Example `meta.json`:

```
{
  "source_url": "https://example.com"
}
```

## URL Rewrite

- `source_url` is taken from `meta.json` when present; otherwise use `home` or `siteurl` from `wp_options` after import.
- `target_url` is the environment primary domain if attached, otherwise the preview URL.
- Replace `source_url` with `target_url` across all tables using a serialized-safe replacement.

## Serialized Data Handling

- URL replacement must preserve PHP serialized data integrity.
- Raw string replacement is not allowed for serialized fields.

## Workflow

1. Download `archive_url`.
2. Validate archive contents (`db.sql`, `files/`).
3. Import database dump into the environment database.
4. Copy `files/` into `wp-content` for the environment.
5. Run URL rewrite with serialized-safe replacement.
6. Run health checks (see docs/health-checks.md).

## Cutover

- Cutover is achieved by attaching or updating the domain only after health checks pass.
- DNS changes are external to Pressluft but should follow a validated import.
