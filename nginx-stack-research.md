# Pressluft NGINX Stack: Gap Analysis & Recommendations

## 1. What You Have Today

Your entire NGINX stack lives in **2 files**:

| File | What it does |
|---|---|
| `ops/ansible/roles/nginx-stack/tasks/main.yml` | Installs packages, creates docroot, deploys config, enables services |
| `ops/ansible/roles/nginx-stack/templates/pressluft-default.conf.j2` | A 21-line server block |

The current server block is essentially a "does NGINX work?" smoke test:

```nginx
server {
    listen 80 default_server;
    server_name _;
    root /srv/www/pressluft/default/public;
    index index.html index.php;
    location / { try_files $uri $uri/ =404; }
    location ~ \.php$ { include snippets/fastcgi-php.conf; fastcgi_pass unix:/run/php/php8.3-fpm.sock; }
    location ~ /\.ht { deny all; }
}
```

**What's missing is basically... everything a WordPress hosting panel needs.**

## 2. Layer-by-Layer Gap Analysis

### Layer A: `nginx.conf` (Main Config)

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| `worker_processes` | distro default (1) | `auto` | **Missing** |
| `worker_rlimit_nofile` | not set | 65535 | **Missing** |
| `worker_connections` | distro default (768) | 2048-4096 | **Missing** |
| `multi_accept` | not set | `on` | **Missing** |
| `use epoll` | not set | `epoll` | **Missing** |
| `sendfile` | not set | `on` | **Missing** |
| `tcp_nopush` | not set | `on` | **Missing** |
| `tcp_nodelay` | not set | `on` | **Missing** |
| `keepalive_timeout` | distro default (65s) | 15-30s | **Missing** |
| `keepalive_requests` | not set | 512-1000 | **Missing** |
| `client_max_body_size` | not set (default 1M) | 64-128M (WP uploads) | **Critical** |
| `client_body_buffer_size` | not set | 16k-128k | **Missing** |
| `server_tokens` | not set (`on`) | `off` | **Security gap** |
| Gzip compression | none | full gzip config | **Major gap** |
| `open_file_cache` | none | configured | **Performance gap** |
| FastCGI cache zone | none | `fastcgi_cache_path` in http block | **Major gap** |
| Rate limit zones | none | `limit_req_zone` for wp-login, xmlrpc | **Security gap** |

**Verdict**: You're running the Ubuntu 24.04 stock `nginx.conf` with zero tuning. Every production WordPress host (SlickStack, WordOps, CloudPanel, SpinupWP) ships a custom `nginx.conf`.

**Recommendation**: Deploy a custom `nginx.conf.j2` template. This is the single highest-impact change.

### Layer B: WordPress Server Block

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| WordPress permalink rewrite | `try_files $uri $uri/ =404` | `try_files $uri $uri/ /index.php?$args` | **WordPress broken** |
| Static asset caching | none | `expires max` on images/css/js/fonts | **Major gap** |
| Block `xmlrpc.php` | no | `deny all` or rate-limit | **Security gap** |
| Block `wp-config.php` | no | `deny all` | **Security gap** |
| Block PHP in uploads | no | `location ~* /uploads/.*\.php$ { deny all; }` | **Security gap** |
| Block dotfiles beyond `.ht` | `.ht` only | all dotfiles (`location ~ /\.`) | **Partial** |
| Rate-limit `wp-login.php` | no | `limit_req zone=wplogin` | **Security gap** |
| `client_max_body_size` per-site | no | 64M+ for media uploads | **Critical** |
| Per-site `access_log` / `error_log` | no | per-site log paths | **Missing** |
| FastCGI cache integration | no | full skip/bypass/serve logic | **Major gap** |
| `add_header X-FastCGI-Cache` | no | cache hit/miss indicator | **Missing** |

**Verdict**: The `=404` fallback means **WordPress pretty permalinks don't work at all**. This is the most critical functional bug. Every single WP hosting panel uses `/index.php?$args` as the fallback.

### Layer C: FastCGI Page Cache

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| `fastcgi_cache_path` | none | `/var/cache/nginx levels=1:2 keys_zone=WORDPRESS:100m inactive=60m max_size=512m` | **Not implemented** |
| `fastcgi_cache_key` | none | `$scheme$request_method$host$request_uri` | **Not implemented** |
| Cache bypass logic | none | Skip for POST, query strings, logged-in users, wp-admin, cookies | **Not implemented** |
| Cache purging | none | nginx-helper plugin + purge module, or stale-while-revalidate | **Not implemented** |
| `X-FastCGI-Cache` header | none | Shows HIT/MISS/BYPASS for debugging | **Not implemented** |

**Verdict**: FastCGI page caching is the single biggest performance lever for WordPress on NGINX. Every serious competitor (SlickStack, WordOps, SpinupWP, CloudPanel) implements it. Without it, every page request hits PHP-FPM + MySQL.

**Recommendation**: This should be a day-1 feature but can be phased. Start with `fastcgi_cache_use_stale` and cookie-based bypass. SlickStack's approach (hardcoded aggressive caching with `inactive=60m`) is a good model for small sites.

### Layer D: PHP-FPM Pool Strategy

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| Pool per site | no (single default pool) | yes -- isolated user, socket, `open_basedir` | **Security gap** |
| `pm` strategy | distro default (`dynamic`) | `ondemand` for shared hosting (many small sites) | **Resource waste** |
| `pm.max_children` | distro default (5) | Calculated: available_RAM / ~40MB per worker | **Not tuned** |
| `pm.process_idle_timeout` | not set | 10-30s for ondemand | **Missing** |
| `pm.max_requests` | distro default (0, unlimited) | 500-1000 (prevents memory leaks) | **Missing** |
| `request_terminate_timeout` | not set | 300s (kill runaway scripts) | **Missing** |
| `php_admin_value[open_basedir]` | not set | `/srv/www/site:/tmp:/usr/share/php` | **Security gap** |
| `expose_php` | default (On) | `Off` | **Info leak** |
| `php_admin_value[disable_functions]` | not set | `exec,passthru,shell_exec,system,proc_open,popen` | **Security gap** |

**Verdict**: For a hosting panel that will run many sites, the pool-per-site model (WordOps, SpinupWP, CloudPanel all do this) is essential for:
- **Security isolation**: Site A can't read Site B's files
- **Resource control**: One runaway site doesn't starve others
- **`pm = ondemand`**: Workers spawn only when needed, idle sites consume zero RAM

**Memory budget formula** (from industry practice):
```text
max_children_per_site = (Total_RAM - OS_overhead - MySQL - Redis - NGINX) / (num_sites * avg_worker_MB)
# For a 4GB VPS with 50 sites: (4096 - 512 - 512 - 128 - 64) / (50 * 40) = ~1.4
# So pm.max_children = 2-3 per site with ondemand
```

### Layer E: Redis Object Cache

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| Redis installed | yes | yes | OK |
| Per-site isolation | none | Key prefix per site (easiest) or `SELECT db` (0-15) | **Missing** |
| `maxmemory` | distro default (no limit) | 128-256MB for shared hosting | **Missing** |
| `maxmemory-policy` | distro default (`noeviction`) | `allkeys-lru` for WP object cache | **Missing** |
| Redis config template | none | Custom `/etc/redis/redis.conf` | **Missing** |
| Unix socket | TCP 6379 (verified in profile) | Unix socket is ~25% faster | **Performance gap** |

**Verdict**: Redis is installed but completely unconfigured. For shared hosting, `allkeys-lru` eviction with a memory cap is essential to prevent one site from consuming all RAM.

### Layer F: SSL/TLS

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| HTTPS support | **none** | Let's Encrypt with auto-renewal | **Critical gap** |
| Certificate provisioning | none | acme.sh (preferred at scale) or certbot | **Not implemented** |
| TLS versions | n/a | TLS 1.2 + 1.3 only | **Not implemented** |
| Cipher suites | n/a | Mozilla Modern or Intermediate | **Not implemented** |
| HSTS | n/a | `max-age=63072000; includeSubDomains; preload` | **Not implemented** |
| OCSP stapling | n/a | `ssl_stapling on` | **Not implemented** |
| HTTP->HTTPS redirect | n/a | 301 redirect server block | **Not implemented** |
| `ssl_session_cache` | n/a | `shared:SSL:10m` | **Not implemented** |
| `ssl_session_tickets` | n/a | `off` (for forward secrecy) | **Not implemented** |

**Verdict**: No SSL at all. The firewall allows port 443 but nothing listens on it. This is expected for an early-stage project, but it's a prerequisite for any production use.

**Recommendation**: acme.sh is preferred over certbot for hosting panels (WordOps, SlickStack both use it) because it's shell-native, supports DNS-01 challenges for wildcards, and doesn't require root. CloudPanel uses certbot. Either works.

### Layer G: Security Headers (NGINX Level)

| Header | Pressluft NGINX | Pressluft Go (control plane) | Industry Standard |
|---|---|---|---|
| `X-Content-Type-Options: nosniff` | no | yes | **Gap at NGINX** |
| `X-Frame-Options` | no | yes (`DENY`) | **Gap at NGINX** |
| `Referrer-Policy` | no | yes (`no-referrer`) | **Gap at NGINX** |
| `Strict-Transport-Security` | no | no | **Gap everywhere** |
| `Permissions-Policy` | no | no | **Gap everywhere** |
| `X-XSS-Protection` | no | no | Most panels add it |
| `Content-Security-Policy` | no | yes (strict) | **Gap at NGINX** |
| `server_tokens off` | no | n/a | **Gap** |

**Verdict**: The Go control plane has good security headers for its own API. The NGINX layer that actually serves WordPress sites has none. SlickStack and WordOps both set these globally in `nginx.conf`.

### Layer H: Logging & Monitoring

| Concern | Pressluft Now | Industry Standard | Gap |
|---|---|---|---|
| Per-site access logs | no | `/var/log/nginx/site.access.log` | **Missing** |
| Per-site error logs | no | `/var/log/nginx/site.error.log` | **Missing** |
| Log rotation | distro default | `logrotate.d/nginx` with daily/14-day retention | **Untested** |
| `stub_status` | no | `/nginx_status` (IP-restricted) for monitoring | **Missing** |
| PHP-FPM status | no | `pm.status_path = /fpm-status` (IP-restricted) | **Missing** |
| PHP-FPM slow log | no | `slowlog` + `request_slowlog_timeout = 5s` | **Missing** |

## 3. Competitor Comparison

| Feature | SlickStack | WordOps | CloudPanel | SpinupWP | Pressluft |
|---|---|---|---|---|---|
| **Custom nginx.conf** | Yes (500+ lines, heavily tuned) | Yes (custom Nginx build) | Yes (Nginx 1.28 + PageSpeed) | Yes (modular conf.d structure) | **No** |
| **WP server block** | Full (hardcoded) | Full (generated per site) | Full (generated per site) | Full (modular includes) | **Broken** (=404) |
| **FastCGI cache** | Yes (aggressive, always-on) | Yes (per-site opt-in: `--wpfc`) | Via Varnish 7.5 | Yes (per-site) | **No** |
| **PHP-FPM pools** | Single pool (single-site only) | Per-site pools | Per-site pools | Per-site pools | **Single default** |
| **PM strategy** | `ondemand` | `ondemand` | `ondemand` | `ondemand` | **distro default** |
| **SSL/TLS** | Let's Encrypt + OpenSSL | acme.sh (A+ SSLLabs) | certbot | certbot | **None** |
| **Redis** | Yes + object cache plugin | Yes + key prefix isolation | Yes | Yes | **Installed, unconfigured** |
| **Security headers** | HSTS, X-Frame, nosniff, XSS | HSTS, X-Frame, nosniff | Yes | Yes | **None at NGINX** |
| **WP security blocks** | xmlrpc, wp-config, uploads, login | xmlrpc, wp-config, uploads, login | Yes | Yes | **None** |
| **Rate limiting** | Yes (per-zone: login, search, PHP, ajax) | Yes | Yes | Yes | **None** |
| **Gzip** | Yes (comprehensive type list) | Yes + Brotli | Yes | Yes | **No** |
| **Per-site logging** | Yes | Yes | Yes | Yes | **No** |
| **Monitoring** | Netdata integration | Netdata + ngx_vts | Built-in dashboard | Built-in | Agent CPU/mem only |
| **Multi-site capable** | No (single-site per server) | Yes (dozens per server) | Yes | Yes | **Intent: yes, config: no** |
| **Automation** | Bash cron scripts | Python CLI | Panel UI | Panel UI | Ansible + Go |

## 4. Recommended Priority Roadmap for Pressluft

Given this is an early-stage project, here's a phased plan ranked by impact:

### Phase 1: Make WordPress Actually Work (Critical)
1. **Fix the `try_files` directive** -- change `=404` to `/index.php?$args`
2. **Deploy a custom `nginx.conf.j2`** with: `worker_processes auto`, `sendfile on`, `tcp_nopush on`, `tcp_nodelay on`, `keepalive_timeout 30`, `client_max_body_size 64m`, `server_tokens off`, gzip compression, `open_file_cache`
3. **Add WordPress security locations** to the per-site server block template: block `wp-config.php`, `xmlrpc.php`, PHP in uploads, dotfiles

### Phase 2: Security & SSL (High)
4. **SSL/TLS template** with Let's Encrypt provisioning (acme.sh or certbot), HSTS, TLS 1.2+1.3, OCSP stapling
5. **Security headers** in `nginx.conf`: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`
6. **Rate limiting zones** for `wp-login.php` and `xmlrpc.php`
7. **PHP hardening**: `expose_php = Off`, `disable_functions`, `open_basedir`

### Phase 3: Performance (High)
8. **FastCGI page cache** with cookie-based bypass for logged-in users
9. **Redis configuration**: `maxmemory 256mb`, `maxmemory-policy allkeys-lru`, Unix socket
10. **Static asset caching**: `expires max` on images/css/js/fonts with `access_log off`

### Phase 4: Multi-Tenancy (Required for Hosting Panel)
11. **Per-site PHP-FPM pools**: separate Unix sockets, `pm = ondemand`, `pm.max_children` based on RAM budget, `open_basedir` isolation
12. **Per-site server block generation**: Jinja2 template that takes domain, docroot, SSL paths, PHP socket as variables
13. **Per-site logging**: separate access/error log paths
14. **Redis isolation**: key prefix per site via `WP_REDIS_PREFIX`

### Phase 5: Observability
15. **NGINX `stub_status`** endpoint for agent to scrape
16. **PHP-FPM status page** (IP-restricted)
17. **PHP-FPM slow log** for debugging

## 5. Recommended "Gold Standard" Config Summary

For a 2-4GB VPS hosting 20-50 small WordPress sites:

| Layer | Setting | Value |
|---|---|---|
| **nginx.conf** | `worker_processes` | `auto` |
|  | `worker_rlimit_nofile` | `65535` |
|  | `worker_connections` | `4096` |
|  | `multi_accept` | `on` |
|  | `sendfile` / `tcp_nopush` / `tcp_nodelay` | all `on` |
|  | `keepalive_timeout` | `30s` |
|  | `keepalive_requests` | `512` |
|  | `client_max_body_size` | `64m` |
|  | `server_tokens` | `off` |
|  | `gzip` | `on`, level 4, comprehensive type list |
|  | `open_file_cache` | `max=10000 inactive=20s` |
| **PHP-FPM** | `pm` | `ondemand` |
|  | `pm.max_children` | 2-4 per site (RAM-dependent) |
|  | `pm.process_idle_timeout` | `10s` |
|  | `pm.max_requests` | `500` |
|  | `request_terminate_timeout` | `300s` |
| **FastCGI Cache** | `keys_zone` | `100m` (~800K cached pages) |
|  | `max_size` | `512m` |
|  | `inactive` | `60m` |
| **Redis** | `maxmemory` | `256mb` |
|  | `maxmemory-policy` | `allkeys-lru` |
|  | transport | Unix socket |
| **SSL** | provider | acme.sh with DNS-01 or HTTP-01 |
|  | protocols | TLS 1.2 + TLS 1.3 |
|  | HSTS | `max-age=63072000` |

This is where Pressluft stands. The foundation (Ansible automation, profile system, agent, verification checks) is solid engineering. The NGINX/PHP-FPM configuration is a placeholder that needs to evolve through the phases above.
