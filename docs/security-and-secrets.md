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
- S3-compatible storage credentials are stored encrypted at rest.
- Database credentials for environments are stored as secrets and injected into `wp-config.php`.
- DNS provider API credentials for ACME DNS-01 (wildcard preview cert) are stored encrypted. See `docs/domain-and-routing.md`.
- Secrets are stored in an encrypted local store at `/var/lib/pressluft/secrets`.

## Audit Logging

- All mutating actions are logged with: user_id, action, resource_id, timestamp, result.

## Node Security Hardening

Every managed node receives a security baseline during provisioning. These measures are always-on and apply uniformly to all environments hosted on the node. See `docs/provisioning-spec.md` for full implementation details.

Summary of layers:

| Layer | Mechanism | Scope |
|-------|-----------|-------|
| Network | Firewall: allow 22, 80, 443 only; deny all other inbound | Node-wide |
| Brute-force protection | Fail2Ban with WordPress-specific jails (login, XML-RPC, scanners) | Node-wide |
| Web application firewall | 7G WAF rules in every Nginx server block | Node-wide |
| PHP hardening | `disable_functions`, `open_basedir`, `expose_php = Off` | Per-pool / global |
| HTTP security headers | `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy` | Node-wide |
| Process isolation | Dedicated Linux user + PHP-FPM pool per environment | Per-environment |
| Filesystem isolation | `open_basedir` restricts PHP to environment directory | Per-environment |
| Upload protection | `shared/uploads` is writable and non-executable | Per-environment |
| Default vhost | Nginx returns 444 for unmatched hostnames | Node-wide |

## Magic Login

Magic login generates a one-time WordPress admin login URL, allowing the Pressluft admin to access any managed environment's WordPress dashboard without knowing the WordPress credentials.

### Mechanism

Magic login is implemented as a **node query** (see `docs/technical-architecture.md` Section 7): a direct SSH command from the Go binary to the target node, bypassing the job queue and Ansible. This is appropriate because magic login is non-destructive, single-command, and time-sensitive.

Execution flow:

1. The API handler receives `POST /api/environments/:id/magic-login` (see `docs/api-contract.md`).
2. The Go binary opens an SSH connection to the environment's node.
3. It executes `wp eval '...'` as the environment's dedicated Linux user (not root), scoped to that environment's WordPress installation at `/var/www/sites/<site_id>/current`.
4. The `wp eval` script uses WordPress's `WP_Session_Tokens` API to create a session token for the first administrator user (user ID 1 or the first user with the `administrator` role).
5. The script sets the token expiration to 60 seconds and outputs the session cookie name and value.
6. The Go binary constructs a login URL using the environment's primary domain (or preview URL) and returns it to the API caller.

### Security Constraints

- **Token lifetime:** 60 seconds from creation. After this, the token cannot be used to establish a new session. Once the token is used and WordPress sets the session cookies, the resulting WordPress session persists with WordPress's normal session lifetime.
- **Scope:** The SSH command runs as the environment's dedicated Linux user with `open_basedir` restrictions. It cannot access other environments.
- **Authentication required:** The API endpoint requires an active Pressluft admin session. No unauthenticated access.
- **Audit logged:** Every magic login generates an audit log entry: `action = magic_login`, `resource_type = environment`, `resource_id = {environment_id}`.
- **No user input interpolation:** The `wp eval` script content is hardcoded in the Go binary. No user-supplied values are interpolated into the PHP code. The environment ID and WordPress path are passed as shell arguments, not embedded in PHP.
- **Timeout:** The SSH command has a hard 10-second timeout (standard for all node queries). If WordPress or WP-CLI is unresponsive, the connection is terminated and an error is returned.

## Transport

- HTTPS enforced for UI/API when `control_plane_domain` is configured. Before a domain is set, the control plane is accessible via `http://<ip>:<port>`. See `docs/domain-and-routing.md`.
- SSH connections use key-based auth only.
