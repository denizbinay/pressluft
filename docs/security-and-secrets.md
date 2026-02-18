# Security and Secrets

This document defines MVP security assumptions and secret handling.

## Authentication

- Single admin user.
- Passwords stored as strong salted hashes.
- Session tokens are random, unguessable, and expire after 24 hours.

## Authorization

- All API endpoints require admin session.
- No multi-user authorization in MVP.

## Secrets

- SSH keys for node access are generated and stored by the control plane.
- S3 storage credentials are stored encrypted at rest.
- Database credentials for environments are stored as secrets and injected into `wp-config.php`.

## Audit Logging

- All mutating actions are logged with: user_id, action, resource_id, timestamp, result.

## Transport

- HTTPS enforced for UI/API.
- SSH connections use key-based auth only.
