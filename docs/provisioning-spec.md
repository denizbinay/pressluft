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

- ACME client required for LetsEncrypt HTTP-01.
- Nginx must expose `/.well-known/acme-challenge/` for all domains.

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
