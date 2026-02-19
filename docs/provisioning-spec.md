# Provisioning Spec

This document defines the MVP provisioning contract. Exact package versions are pinned in Ansible; this spec defines minimum supported versions and required configuration.

## Supported OS

- Ubuntu 24.04 LTS only.

## Required Components

- Nginx (min 1.22)
- PHP-FPM (min 8.2), with `php-redis` extension
- MariaDB (min 10.11)
- Redis (min 7.0)
- WP-CLI (latest stable)
- Fail2Ban (min 1.0)
- rsync
- unzip
- curl

## TLS

- ACME client (certbot) required for Let's Encrypt certificate management.
- Nginx must expose `/.well-known/acme-challenge/` for all domains (HTTP-01 challenges).
- Custom domains use HTTP-01 challenge (per-domain certificate).
- Preview wildcard certificate uses DNS-01 challenge when operator configures `preview_domain` and `dns01_provider`. Requires the appropriate certbot DNS plugin (e.g., `certbot-dns-cloudflare`, `certbot-dns-hetzner`). See `docs/domain-and-routing.md`.
- certbot systemd timer handles automatic renewal for all certificates (both HTTP-01 and DNS-01). A post-renewal hook reloads Nginx.
- Provisioning installs certbot and the DNS plugin specified by `dns01_provider` (if configured).

## Firewall and Ports

- Allow inbound: 22/tcp, 80/tcp, 443/tcp.
- Deny all other inbound by default.

## System Users

- `pressluft` system user for control plane service.
- Per-environment Linux users created at site/env creation.

## File Permissions

- `/var/www/sites` owned by `pressluft`.
- Environment directories owned by their dedicated user.
- `shared/uploads` is writable and non-executable.

## SSH

- Control plane uses SSH to manage nodes (including localhost).
- SSH keys are generated and stored by the control plane.

## Redis

Redis provides object caching for WordPress environments. One Redis instance per node, shared by all environments on that node.

Configuration:

- Bind to `127.0.0.1` only (no network exposure).
- Disable persistence (`save ""`) — Redis is used as a volatile cache, not a data store. No RDB or AOF.
- Set `maxmemory` to 10% of system RAM (detected at provisioning time) with `allkeys-lru` eviction policy.
- Enable as a systemd service (`redis-server.service`).
- Default port 6379 (localhost only, not exposed by firewall).

Isolation between environments is prefix-based: each environment uses the Redis key prefix `pressluft_{environment_id}:`. The Redis database number is always 0. This is configured per-environment in `wp-config.php` via the `WP_REDIS_PREFIX` constant.

## FastCGI Cache

Nginx FastCGI page caching is configured at the node level during provisioning:

- Cache directory: `/var/cache/nginx/fastcgi`, owned by `www-data`.
- Global Nginx config includes: `fastcgi_cache_path /var/cache/nginx/fastcgi levels=1:2 keys_zone=WORDPRESS:100m inactive=60m max_size=512m;`

Per-environment enablement is controlled by the `fastcgi_cache_enabled` column on the `environments` table (see `docs/data-model.md`). The Nginx server block template conditionally includes cache directives based on this setting. See `docs/domain-and-routing.md` for the full caching configuration within server blocks.

## Security Hardening

Provisioning enforces a security baseline on every node. These measures are always-on and apply uniformly to all environments. There are no per-environment security toggles.

### Fail2Ban

Fail2Ban monitors log files for malicious patterns and bans offending IPs via iptables.

Configuration file: `/etc/fail2ban/jail.d/pressluft.conf`

| Jail | Watches | Threshold | Ban Duration | Notes |
|------|---------|-----------|-------------|-------|
| `wordpress-login` | Nginx access log — repeated POST to `/wp-login.php` | 5 failures in 10 min | 1 hour | Brute-force login protection |
| `wordpress-xmlrpc` | Nginx access log — POST to `/xmlrpc.php` | 3 attempts in 1 min | 24 hours | XML-RPC is almost exclusively abused; legitimate use is rare |
| `nginx-http-auth` | Nginx error log — HTTP auth failures | 5 failures in 10 min | 1 hour | Catches basic auth brute-force |
| `nginx-botsearch` | Nginx access log — scanner-pattern 404s | 10 hits in 10 min | 1 hour | Blocks vulnerability scanners |

Common settings:

- Ban action: `iptables-multiport` (blocks ports 80, 443).
- `ignoreip = 127.0.0.1/8` — never ban localhost.
- Log paths: derived from Nginx access and error log locations on the node.
- Fail2Ban enabled as a systemd service.

### 7G Web Application Firewall

The 7G Firewall (by Jeff Starr / Perishable Press) is a set of Nginx rewrite rules that block known malicious request patterns at the web server level, before requests reach PHP.

Rules are **vendored** in the Ansible role `nginx-waf` as a static file and updated with Pressluft releases. They are not pulled from external URLs at runtime.

Deployment:

- Installed to `/etc/nginx/conf.d/7g-firewall.conf` during node provisioning.
- Included in every Nginx server block via `include /etc/nginx/conf.d/7g-firewall.conf;` (see `docs/domain-and-routing.md`).
- An exclusion file at `/etc/nginx/conf.d/7g-exclusions.conf` is created empty during provisioning. Operators can add custom exclusion rules here to handle false positives (e.g., WooCommerce REST API paths, unusual plugin URL patterns). This file is included after the main 7G rules.

Coverage:

- Bad query strings (SQL injection patterns, path traversal, etc.)
- Bad request URIs (admin probes, backup file access, etc.)
- Bad user agents (known scanners, scrapers, attack tools)
- Bad referrers (spam referrer patterns)
- Bad request methods (TRACE, DELETE, TRACK, etc. — only GET, POST, HEAD allowed)
- Bad HTTP headers (null bytes, oversized headers)

### PHP Hardening

Applied to the global `php.ini` (affects all PHP-FPM pools on the node):

| Directive | Value | Rationale |
|-----------|-------|-----------|
| `disable_functions` | `exec,passthru,shell_exec,system,proc_open,popen` | Prevents PHP code from spawning processes. WP-CLI runs as a separate process outside PHP-FPM, so these are safe to disable. |
| `expose_php` | `Off` | Do not reveal PHP version in HTTP headers. |
| `allow_url_include` | `Off` | Prevents remote file inclusion attacks. |
| `allow_url_fopen` | `On` | Required by WordPress for plugin/theme updates and external HTTP requests. |

Applied per PHP-FPM pool (set in the pool configuration, not global php.ini):

| Directive | Value | Rationale |
|-----------|-------|-----------|
| `open_basedir` | `/var/www/sites/<site_id>/:/tmp/:/usr/share/php/` | Restricts file access to the environment's directory, temp files, and PHP system libraries. Prevents a compromised plugin from reading other environments' files. |

### Nginx Security Headers

Applied globally in the Nginx `http` block (affects all server blocks on the node):

| Header | Value | Notes |
|--------|-------|-------|
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing. |
| `X-Frame-Options` | `SAMEORIGIN` | Prevents clickjacking. |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limits referrer information leakage. |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` | Disables browser APIs not needed by WordPress. |

HSTS (`Strict-Transport-Security`) is **not** applied globally because not all environments have TLS (sslip.io preview URLs are served over plain HTTP). HSTS would break HTTP-only preview access.

## Idempotency

- Provisioning is safe to run multiple times.
- Playbooks must detect existing configuration and skip safely.

## Observability

- Logs stored in `/var/log/pressluft/`.
- Control plane exposes basic metrics at `GET /api/metrics`.
