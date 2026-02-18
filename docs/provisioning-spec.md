# Provisioning Spec

This document defines the MVP provisioning contract. Exact package versions are pinned in Ansible; this spec defines minimum supported versions and required configuration.

## Supported OS

- Ubuntu 24.04 LTS only.

## Required Components

- Nginx (min 1.22)
- PHP-FPM (min 8.2)
- MariaDB (min 10.11)
- WP-CLI (latest stable)
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

## Idempotency

- Provisioning is safe to run multiple times.
- Playbooks must detect existing configuration and skip safely.

## Observability

- Logs stored in `/var/log/pressluft/`.
- Control plane exposes basic metrics at `GET /api/metrics`.
